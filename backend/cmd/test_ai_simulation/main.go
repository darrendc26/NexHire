package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/genai"

	"nexhire/backend/ai"
	"nexhire/backend/auth"
	"nexhire/backend/candidate"
	"nexhire/backend/interview"
	"nexhire/backend/models"
)

func main() {
	// 1. Load .env config
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatalf("❌ GEMINI_API_KEY is not set in environment or .env file.")
	}

	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-3.5-flash-lite"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5433/nexhire?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping DB: %v", err)
	}

	fmt.Printf("🚀 Starting Live Gemini AI Interview Simulation\n")
	fmt.Printf("🤖 Model: %s\n", modelName)
	fmt.Printf("🐘 DB Connected: %s\n\n", dbURL)

	ctx := context.Background()
	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		log.Fatalf("❌ Failed to create GenAI client: %v", err)
	}

	aiProvider := ai.NewGeminiProvider(genaiClient, modelName)
	aiService := ai.NewService(aiProvider)

	interviewRepo := interview.NewPostgresRepository(db)
	candidateRepo := candidate.NewPostgresRepository(db)

	authSvc := auth.NewServiceWithDB(auth.Load(), db)
	testRecruiter, err := authSvc.GetOrCreateUser(ctx, &models.GoogleUserInfo{
		Sub:     "sim_recruiter_sub_123",
		Email:   "sim.recruiter@nexhire.com",
		Name:    "Simulation Recruiter",
		Picture: "https://example.com/pic.png",
	})
	if err != nil {
		log.Fatalf("❌ Failed to create test recruiter: %v", err)
	}

	interviewSvc := interview.NewService(interviewRepo)
	candidateSvc := candidate.NewService(candidateRepo, aiService)

	// 2. Create Interview (5 min duration)
	newInterview, err := interviewSvc.Create(ctx, testRecruiter.ID, interview.CreateInterviewRequest{
		Title:        "Senior Go Backend Engineer",
		Role:         "Backend Engineer",
		Description:  "Golang microservices and high-throughput systems",
		Difficulty:   models.Medium,
		Duration:     5,
		VoiceEnabled: true,
	})
	if err != nil {
		log.Fatalf("❌ Failed to create interview: %v", err)
	}

	fmt.Printf("📋 Interview Created: ID=%s, Title=%q, Duration=%d min, ShareToken=%s\n",
		newInterview.ID, newInterview.Title, newInterview.Duration, newInterview.ShareToken)

	// 3. Start Candidate Session
	session, err := candidateSvc.StartSession(ctx, newInterview.ShareToken, candidate.StartSessionRequest{
		Name:  "Alex Miller",
		Email: "alex.miller@example.com",
	})
	if err != nil {
		log.Fatalf("❌ Failed to start candidate session: %v", err)
	}

	rawToken := session.RawToken
	fmt.Printf("👤 Candidate Session Started: Candidate=%s, SessionID=%s\n\n", session.Name, session.ID)

	// 4. Turn 1: Start Session Question (Gemini initial question)
	fmt.Println("----------------------------------------------------------------------")
	fmt.Println("🔹 TURN 1: Generating Initial Question from Gemini LLM...")
	startResp, err := candidateSvc.StartSessionQuestion(ctx, rawToken)
	if err != nil {
		log.Fatalf("❌ StartSessionQuestion failed: %v", err)
	}

	fmt.Printf("❓ Q1 [%s] (Order %d): %s\n", startResp.Question.ID, startResp.Question.Order, startResp.Question.Text)
	fmt.Printf("⏱️  Time Remaining: %d seconds\n\n", startResp.Progress.TimeRemaining)

	// Turn 1 Answer: PERFECT URL shortener answer with realistic spoken speech blemishes
	answer1 := "Um... for a URL shortener with 10k reads/sec and 100 writes/sec... uh, I'd choose PostgreSQL as the primary transactional database using a Base62 encoded auto-increment ID column. For low-latency reads, um, I'd put Redis in front as a Cache-Aside layer caching short-code to long-URL mappings. Since 10,000 reads/sec is high, Redis handles 99% of read requests in under 1ms, while PostgreSQL easily handles 100 writes/sec."
	fmt.Printf("💬 Candidate Answer 1 (PERFECT - Spoken style):\n%s\n\n", answer1)

	fmt.Println("🤖 Sending Answer 1 to Gemini for evaluation...")
	ans1Resp, err := candidateSvc.SubmitAnswer(ctx, rawToken, answer1)
	if err != nil {
		log.Fatalf("❌ SubmitAnswer 1 failed: %v", err)
	}

	if ans1Resp.Evaluation != nil {
		fmt.Printf("⭐ Gemini Score: %.1f / 10.0\n", ans1Resp.Evaluation.Score)
		fmt.Printf("📝 Gemini Feedback: %s\n", ans1Resp.Evaluation.Feedback)
		fmt.Printf("💪 Strengths Identified: %v\n", ans1Resp.Evaluation.Strengths)
		fmt.Printf("⚠️  Weaknesses Identified: %v\n", ans1Resp.Evaluation.Weaknesses)
	}

	if !ans1Resp.ShouldContinue || ans1Resp.NextQuestion == nil {
		fmt.Println("🏁 Interview finished after Turn 1.")
		printReport(ctx, candidateSvc, rawToken)
		return
	}

	// 5. Turn 2: IMPERFECT / SHALLOW Answer with spoken blemishes
	fmt.Println("\n----------------------------------------------------------------------")
	fmt.Printf("🔹 TURN 2: Next Question from Gemini...\n")
	fmt.Printf("❓ Q2 [%s] (Order %d): %s\n", ans1Resp.NextQuestion.ID, ans1Resp.NextQuestion.Order, ans1Resp.NextQuestion.Text)
	fmt.Printf("⏱️  Time Remaining: %d seconds\n\n", ans1Resp.Progress.TimeRemaining)

	answer2 := "Uh, yeah... so... B-Tree indexes speed up SELECT queries by storing keys in a balanced tree structure with O(log N) lookup complexity. But, uh... writes can get a bit slower because, you know, the database has to update the index nodes when inserting or updating rows. I, uh... haven't configured composite index ordering or fill-factor tuning extensively, but um... that's the main trade-off."
	fmt.Printf("💬 Candidate Answer 2 (IMPERFECT - Spoken style):\n%s\n\n", answer2)

	fmt.Println("🤖 Sending Answer 2 to Gemini for evaluation...")
	ans2Resp, err := candidateSvc.SubmitAnswer(ctx, rawToken, answer2)
	if err != nil {
		log.Fatalf("❌ SubmitAnswer 2 failed: %v", err)
	}

	if ans2Resp.Evaluation != nil {
		fmt.Printf("⭐ Gemini Score: %.1f / 10.0\n", ans2Resp.Evaluation.Score)
		fmt.Printf("📝 Gemini Feedback: %s\n", ans2Resp.Evaluation.Feedback)
		fmt.Printf("💪 Strengths Identified: %v\n", ans2Resp.Evaluation.Strengths)
		fmt.Printf("⚠️  Weaknesses Identified: %v\n", ans2Resp.Evaluation.Weaknesses)
	}

	if !ans2Resp.ShouldContinue || ans2Resp.NextQuestion == nil {
		fmt.Println("🏁 Interview finished after Turn 2.")
		printReport(ctx, candidateSvc, rawToken)
		return
	}

	// 6. Turn 3: SOMEWHAT CORRECT Answer with spoken blemishes
	fmt.Println("\n----------------------------------------------------------------------")
	fmt.Printf("🔹 TURN 3: Next Question from Gemini...\n")
	fmt.Printf("❓ Q3 [%s] (Order %d): %s\n", ans2Resp.NextQuestion.ID, ans2Resp.NextQuestion.Order, ans2Resp.NextQuestion.Text)
	fmt.Printf("⏱️  Time Remaining: %d seconds\n\n", ans2Resp.Progress.TimeRemaining)

	answer3 := "Well, um... in Go, for handling race conditions and concurrent access to shared resources... uh, you can use channels or sync.Mutex. Channels are great for passing ownership of data between goroutines—like, you know, 'don't communicate by sharing memory, share memory by communicating'. But uh, for a simple shared counter or map in a service... um, a RWMutex is often faster and simpler, although... uh, if you forget to release the lock or hold it too long, it can cause thread contention under heavy load."
	fmt.Printf("💬 Candidate Answer 3 (SOMEWHAT CORRECT - Spoken style):\n%s\n\n", answer3)

	// Update DB session expires_at to 30s remaining so SubmitAnswer triggers final candidate report synthesis
	_, _ = db.ExecContext(ctx, "UPDATE candidate_sessions SET expires_at = $1 WHERE id = $2", time.Now().Add(30*time.Second), session.ID)

	fmt.Println("🤖 Sending Answer 3 to Gemini for final evaluation and Report Synthesis...")
	ans3Resp, err := candidateSvc.SubmitAnswer(ctx, rawToken, answer3)
	if err != nil {
		log.Fatalf("❌ SubmitAnswer 3 failed: %v", err)
	}

	if ans3Resp.Evaluation != nil {
		fmt.Printf("⭐ Gemini Score: %.1f / 10.0\n", ans3Resp.Evaluation.Score)
		fmt.Printf("📝 Gemini Feedback: %s\n", ans3Resp.Evaluation.Feedback)
		fmt.Printf("💪 Strengths Identified: %v\n", ans3Resp.Evaluation.Strengths)
		fmt.Printf("⚠️  Weaknesses Identified: %v\n", ans3Resp.Evaluation.Weaknesses)
	}

	fmt.Printf("\n🏁 Interview Completed! Outro: %s\n", ans3Resp.OutroMessage)

	// 7. Print Synthesized Executive Candidate Report from Gemini
	printReport(ctx, candidateSvc, rawToken)
}

func printReport(ctx context.Context, candidateSvc *candidate.Service, rawToken string) {
	fmt.Println("\n======================================================================")
	fmt.Println("📊 SYNTHESIZED CANDIDATE EVALUATION REPORT (Generated by Gemini LLM)")
	fmt.Println("======================================================================")

	time.Sleep(1 * time.Second)
	report, err := candidateSvc.GetReport(ctx, rawToken)
	if err != nil {
		log.Fatalf("❌ Failed to fetch final report: %v", err)
	}

	fmt.Printf("🏆 Overall Score: %.1f / 100.0\n", report.OverallScore)
	fmt.Printf("🎯 Recommendation: %s\n\n", report.Recommendation)
	fmt.Printf("📌 Executive Summary:\n%s\n\n", report.Summary)

	fmt.Println("💪 Demonstrated Strengths:")
	for _, s := range report.Strengths {
		fmt.Printf("  • %s\n", s)
	}

	fmt.Println("\n⚠️  Identified Weaknesses:")
	for _, w := range report.Weaknesses {
		fmt.Printf("  • %s\n", w)
	}

	fmt.Println("\n🧠 Evaluated Skill Breakdown:")
	for _, sk := range report.Skills {
		fmt.Printf("  • %s: Score %.1f/100 — %s\n", sk.Skill, sk.Score, sk.Feedback)
	}

	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	fmt.Printf("\n📄 Full Report JSON:\n%s\n", string(reportJSON))
}
