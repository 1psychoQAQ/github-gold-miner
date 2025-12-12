package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github-gold-miner/internal/adapter/gemini" // 引入 gemini
	"github-gold-miner/internal/adapter/github"
	"github-gold-miner/internal/adapter/repository"
)

func main() {
	// 获取环境变量
	githubToken := os.Getenv("GITHUB_TOKEN")
	geminiKey := os.Getenv("GEMINI_API_KEY") // 新增

	// 数据库连接
	dsn := "host=localhost user=postgres password=123456 dbname=gold_miner port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	ctx := context.Background()

	// 1. 初始化依赖
	repoStore, err := repository.NewPostgresRepo(dsn)
	if err != nil {
		log.Fatalf("❌ DB 初始化失败: %v", err)
	}

	scouter := github.NewScouter(githubToken)

	// 初始化 AI 鉴定师
	appraiser, err := gemini.NewGeminiAppraiser(ctx, geminiKey)
	if err != nil {
		log.Fatalf("❌ AI 初始化失败: %v", err)
	}

	fmt.Println("🚀 开始搜寻金矿 (Go + AI)...")

	// 2. 搜寻 (Scout)
	repos, err := scouter.Scout(ctx, "Go")
	if err != nil {
		log.Fatalf("❌ 搜寻失败: %v", err)
	}
	fmt.Printf("🔍 发现 %d 个项目，开始 AI 鉴定...\n", len(repos))

	// 3. AI 鉴定 + 入库 (Loop)
	for i, r := range repos {
		fmt.Printf("[%d/%d] 正在分析: %s ... ", i+1, len(repos), r.Name)

		exists, _ := repoStore.Exists(ctx, r.ID)
		if exists {
			fmt.Println("⏭️  已存在，跳过")
			continue
		}

		// 调用 AI 进行鉴定
		analyzedRepo, err := appraiser.Appraise(ctx, r)

		// 【修改点】即使 err != nil，analyzedRepo 现在也不是 nil 了，可以安全使用
		if err != nil {
			fmt.Printf("⚠️  AI 分析失败 (将只保存基础信息): %v\n", err)
			// 这里我们不再 continue，而是继续往下走，去保存基础信息
		}

		// 存入数据库
		// 因为我们在 adapter 里修复了 bug，这里 analyzedRepo 绝对不会是 nil
		if err := repoStore.Save(ctx, analyzedRepo); err != nil {
			log.Printf("❌ 保存失败: %v", err)
		} else {
			if err == nil { // 只有 AI 成功了才打印这一段
				fmt.Printf("\n    💰 商业分: %d | 🧠 学习分: %d\n", analyzedRepo.CommercialScore, analyzedRepo.EducationalScore)
				fmt.Printf("    🤖 简评: %s\n", analyzedRepo.Summary)
			} else {
				fmt.Println("    💾 基础信息已保存 (无 AI 分析)")
			}
		}

		time.Sleep(4 * time.Second)
	}
}
