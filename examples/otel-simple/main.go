package main

import (
	"context"
	"embed"
	"time"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	_ "github.com/bronystylecrazy/ultrastructure/examples/otel-simple/docs"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	web "github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type apiService struct {
	otel.Telemetry

	name          string
	workerService *workerService
	db            *gorm.DB
}

func NewAPIService(workerService *workerService, db *gorm.DB) *apiService {
	return &apiService{Telemetry: otel.Nop(), db: db, name: "api", workerService: workerService}
}

func (s *apiService) Start(ctx context.Context) error {
	ctx, obs := s.Obs.Start(ctx, "service.start")
	defer obs.End()

	s.Obs.Info("Starting", zap.String("service", s.name))
	time.Sleep(20 * time.Millisecond)

	return s.workerService.Foo(ctx, "from api service")
}

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Email     string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	CreatedAt time.Time `gorm:"type:timestamp;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;not null;default:now()" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

func (h *apiService) Handle(r fiber.Router) {
	r.Get("/api", func(c fiber.Ctx) error {
		ctx, obs := h.Obs.Start(c.Context(), "api called")
		defer obs.End()

		newUser := &User{
			Email: uuid.NewString() + "test@example.com",
			Name:  "Test User",
		}

		tx := h.db.WithContext(ctx).Create(newUser)
		if tx.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": tx.Error.Error(),
			})
		}

		return c.JSON(newUser)
	})
}

func (s *apiService) Stop(ctx context.Context) error {
	ctx, obs := s.Obs.Start(ctx, "service.stop")
	defer obs.End()

	obs.Info("Stopping", zap.String("service", s.name))
	time.Sleep(10 * time.Millisecond)
	return nil
}

type workerService struct {
	otel.Telemetry
}

func NewWorkerService() *workerService {
	return &workerService{}
}

func (s *workerService) Foo(ctx context.Context, arg string) error {
	ctx, obs := s.Obs.Start(ctx, "worker.foo")
	defer obs.End()

	obs.Info("Foo called", zap.String("arg", arg))
	time.Sleep(5 * time.Millisecond)

	return s.Bar(ctx, arg)
}

func (s *workerService) Bar(ctx context.Context, arg string) error {
	ctx, obs := s.Obs.Start(ctx, "worker.bar")
	defer obs.End()

	obs.Info("Bar called", zap.String("arg", arg))
	time.Sleep(5 * time.Millisecond)
	return nil
}

//go:embed all:web/dist
var assets embed.FS

//go:embed all:db/migrations
var migrations embed.FS

func main() {

	options := di.App(
		otel.Module(),
		lifecycle.Module(),
		realtime.Module(),
		database.Module(
			database.UseOtelLogger(),
			database.UseOtelTraceMetrics(),
			database.UseMigrations(&migrations),
		),
		web.Module(
			web.UseMqttWebsocket(),
			web.UseSpa(&assets),
			web.UseSwagger(),

			di.Provide(NewAPIService),
			di.Provide(NewWorkerService),
		),
		di.Invoke(func(db *gorm.DB, logger *zap.Logger) {
			sqdb, err := db.DB()
			if err != nil {
				panic(err)
			}
			var r int
			if err := sqdb.QueryRow(`SELECT 1 * 1 as result;`).Scan(&r); err != nil {
				panic(err)
			}
			logger.Info("Database connection successful", zap.Int("result", r))
		}),
		fx.NopLogger,
	).Build()

	fx.New(options).Run()

}
