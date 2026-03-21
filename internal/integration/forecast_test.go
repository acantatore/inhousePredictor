package integration_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/naranjax/inhousepredictor/internal/forecast"
	"github.com/naranjax/inhousepredictor/internal/market"
	"github.com/naranjax/inhousepredictor/internal/testutil"
	"github.com/naranjax/inhousepredictor/internal/trade"
)

func TestCreateQuestionAutoCreatesShadowMarket(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerID := createUser(t, pool, "owner@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	contributorID := createUser(t, pool, "contrib@example.com", false)

	svc := forecast.NewService(forecast.NewRepository(pool), market.NewRepository(pool), nil)
	question, err := svc.CreateQuestion(context.Background(), forecast.CreateQuestionParams{
		Title:          "Will Payments ship M3 by June 30?",
		Description:    "Roadmap commitment",
		Program:        "payments",
		OwnerID:        ownerID,
		ResolverID:     resolverID,
		ResolutionRule: "Delivered by June 30 with milestone scope complete.",
		ClosesAt:       time.Now().Add(24 * time.Hour),
		ResolvesAt:     time.Now().Add(48 * time.Hour),
		ContributorIDs: []uuid.UUID{contributorID},
	})
	require.NoError(t, err)
	require.NotNil(t, question.LinkedMarketID)

	var isShadow bool
	var forecastQuestionID uuid.UUID
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT is_shadow, forecast_question_id FROM markets WHERE id = $1`, *question.LinkedMarketID).Scan(&isShadow, &forecastQuestionID))
	require.True(t, isShadow)
	require.Equal(t, question.ID, forecastQuestionID)

	var contributorCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM forecast_question_contributors WHERE forecast_question_id = $1`, question.ID).Scan(&contributorCount))
	require.Equal(t, 2, contributorCount)
}

func TestForecastRevisionsAreAppendOnlyAndUpdateProjection(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerID := createUser(t, pool, "owner@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	contributorA := createUser(t, pool, "a@example.com", false)
	contributorB := createUser(t, pool, "b@example.com", false)
	svc := forecast.NewService(forecast.NewRepository(pool), market.NewRepository(pool), nil)
	question, err := svc.CreateQuestion(context.Background(), forecast.CreateQuestionParams{
		Title:          "Will search relevance improve this quarter?",
		Description:    "Forecast wedge",
		Program:        "search",
		OwnerID:        ownerID,
		ResolverID:     resolverID,
		ResolutionRule: "Success means NDCG improves by 5%.",
		ClosesAt:       time.Now().Add(24 * time.Hour),
		ResolvesAt:     time.Now().Add(48 * time.Hour),
		ContributorIDs: []uuid.UUID{contributorA, contributorB},
	})
	require.NoError(t, err)

	_, err = svc.SubmitForecast(context.Background(), forecast.SubmitForecastParams{QuestionID: question.ID, UserID: contributorA, ProbabilityBps: 4000, Rationale: "Need more validation."})
	require.NoError(t, err)
	updated, err := svc.SubmitForecast(context.Background(), forecast.SubmitForecastParams{QuestionID: question.ID, UserID: contributorB, ProbabilityBps: 8000, Rationale: "Signals are strong."})
	require.NoError(t, err)
	require.Equal(t, 6000, updated.Projection.OfficialProbabilityBps)

	updated, err = svc.SubmitForecast(context.Background(), forecast.SubmitForecastParams{QuestionID: question.ID, UserID: contributorA, ProbabilityBps: 7000, Rationale: "New customer evidence."})
	require.NoError(t, err)
	require.Equal(t, 7500, updated.Projection.OfficialProbabilityBps)

	var revisionCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM forecast_revisions WHERE forecast_question_id = $1`, question.ID).Scan(&revisionCount))
	require.Equal(t, 3, revisionCount)

	var latestProbability int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT probability_bps FROM forecast_latest_active WHERE forecast_question_id = $1 AND user_id = $2`, question.ID, contributorA).Scan(&latestProbability))
	require.Equal(t, 7000, latestProbability)
}

func TestShadowMarketPriceDoesNotChangeOfficialForecast(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerID := createUser(t, pool, "owner@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	contributorA := createUser(t, pool, "a@example.com", false)
	contributorB := createUser(t, pool, "b@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)
	svc := forecast.NewService(forecast.NewRepository(pool), market.NewRepository(pool), nil)
	question, err := svc.CreateQuestion(context.Background(), forecast.CreateQuestionParams{
		Title:          "Will infra keep p99 under target?",
		Description:    "Forecast wedge",
		Program:        "infra",
		OwnerID:        ownerID,
		ResolverID:     resolverID,
		ResolutionRule: "p99 under 200ms.",
		ClosesAt:       time.Now().Add(24 * time.Hour),
		ResolvesAt:     time.Now().Add(48 * time.Hour),
		ContributorIDs: []uuid.UUID{contributorA, contributorB},
	})
	require.NoError(t, err)
	_, err = svc.SubmitForecast(context.Background(), forecast.SubmitForecastParams{QuestionID: question.ID, UserID: contributorA, ProbabilityBps: 6000, Rationale: "Current trend stable."})
	require.NoError(t, err)
	updated, err := svc.SubmitForecast(context.Background(), forecast.SubmitForecastParams{QuestionID: question.ID, UserID: contributorB, ProbabilityBps: 7000, Rationale: "Strong mitigation in place."})
	require.NoError(t, err)
	require.Equal(t, 6500, updated.Projection.OfficialProbabilityBps)

	tradeSvc := trade.NewService(trade.NewRepository(pool), nil)
	_, err = tradeSvc.Execute(context.Background(), trade.TradeRequest{UserID: traderID, MarketID: *question.LinkedMarketID, Side: trade.SideYes, Cost: 300})
	require.NoError(t, err)

	afterTrade, err := svc.GetQuestion(context.Background(), question.ID)
	require.NoError(t, err)
	require.Equal(t, 6500, afterTrade.Projection.OfficialProbabilityBps)
}

func TestSubmitForecastRejectsAfterCloseAndScoresUsePreCloseSnapshot(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerID := createUser(t, pool, "owner@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	contributor := createUser(t, pool, "contrib@example.com", false)
	svc := forecast.NewService(forecast.NewRepository(pool), market.NewRepository(pool), nil)
	question, err := svc.CreateQuestion(context.Background(), forecast.CreateQuestionParams{
		Title:          "Will QA exit beta this month?",
		Description:    "Forecast wedge",
		Program:        "qa",
		OwnerID:        ownerID,
		ResolverID:     resolverID,
		ResolutionRule: "Beta exit by month end.",
		ClosesAt:       time.Now().Add(2 * time.Hour),
		ResolvesAt:     time.Now().Add(4 * time.Hour),
		ContributorIDs: []uuid.UUID{contributor},
	})
	require.NoError(t, err)
	_, err = svc.SubmitForecast(context.Background(), forecast.SubmitForecastParams{QuestionID: question.ID, UserID: contributor, ProbabilityBps: 7000, Rationale: "Confidence is rising."})
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), `UPDATE forecast_questions SET closes_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, question.ID)
	require.NoError(t, err)

	_, err = svc.SubmitForecast(context.Background(), forecast.SubmitForecastParams{QuestionID: question.ID, UserID: contributor, ProbabilityBps: 9000, Rationale: "Too late update."})
	require.Error(t, err)

	_, err = pool.Exec(context.Background(), `UPDATE forecast_questions SET status = 'closed' WHERE id = $1`, question.ID)
	require.NoError(t, err)
	resolved, err := svc.ResolveQuestion(context.Background(), question.ID, resolverID, forecast.OutcomeDelivered, "https://example.com/evidence/qa")
	require.NoError(t, err)
	require.Equal(t, forecast.QuestionStatusResolved, resolved.Status)

	questionAfter, err := svc.GetQuestion(context.Background(), question.ID)
	require.NoError(t, err)
	require.Len(t, questionAfter.Scores, 1)
	require.Equal(t, 7000, questionAfter.Scores[0].ProbabilityBps)
	require.InDelta(t, math.Pow(0.7-1.0, 2), questionAfter.Scores[0].BrierScore, 0.0001)
}

func TestReminderJobsAreDedupedAndWorkerCompletesDueJobs(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerID := createUser(t, pool, "owner@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	contributor := createUser(t, pool, "contrib@example.com", false)
	repo := forecast.NewRepository(pool)
	svc := forecast.NewService(repo, market.NewRepository(pool), nil)
	question, err := svc.CreateQuestion(context.Background(), forecast.CreateQuestionParams{
		Title:          "Will notifications behave?",
		Description:    "Job testing",
		Program:        "ops",
		OwnerID:        ownerID,
		ResolverID:     resolverID,
		ResolutionRule: "Any behavior issue counts as miss.",
		ClosesAt:       time.Now().Add(30 * time.Hour),
		ResolvesAt:     time.Now().Add(48 * time.Hour),
		ContributorIDs: []uuid.UUID{contributor},
	})
	require.NoError(t, err)

	var jobCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM background_jobs WHERE kind = 'forecast_reminder' AND status = 'pending'`).Scan(&jobCount))
	require.Equal(t, 2, jobCount)

	require.NoError(t, repo.EnqueueJob(context.Background(), "resolution_notification", map[string]any{"forecast_question_id": question.ID}, time.Now().Add(-time.Minute), "resolution-dedupe"))
	require.NoError(t, repo.EnqueueJob(context.Background(), "resolution_notification", map[string]any{"forecast_question_id": question.ID}, time.Now().Add(-time.Minute), "resolution-dedupe"))
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM background_jobs WHERE dedupe_key = 'resolution-dedupe'`).Scan(&jobCount))
	require.Equal(t, 1, jobCount)

	worker := forecast.NewWorker(repo, nil)
	ctx, cancel := context.WithCancel(context.Background())
	go worker.Run(ctx)
	time.Sleep(3 * time.Second)
	cancel()

	var completed int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM background_jobs WHERE dedupe_key = 'resolution-dedupe' AND status = 'completed'`).Scan(&completed))
	require.Equal(t, 1, completed)
}
