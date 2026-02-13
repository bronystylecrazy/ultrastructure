package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	_ "github.com/bronystylecrazy/ultrastructure/examples/otel-simple/docs"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/multierr"
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

//go:embed all:migrations
var migrations embed.FS

type handler struct {
	otel.Telemetry
}

func NewHandler() *handler {
	return &handler{
		Telemetry: otel.Nop(),
	}
}

func (h *handler) Authorize() fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, obs := h.Obs.Start(c.Context(), "realtime.authorize")
		defer obs.End()

		obs.Info("Authorize called")
		time.Sleep(5 * time.Millisecond)

		c.SetContext(ctx)

		return c.Next()
	}
}

type pingService struct {
	log *zap.Logger
}

func NewPingService(log *zap.Logger) *pingService {
	return &pingService{log: log}
}

func (p *pingService) Ping(ctx context.Context, target string) error {
	p.log.Info("ping", zap.String("target", target))
	fmt.Println("pong:", target)
	return nil
}

type serveCommand struct {
	log *zap.Logger
}

func NewServeCommand(log *zap.Logger) *serveCommand {
	return &serveCommand{log: log}
}

func (s *serveCommand) Command() *cobra.Command {
	return &cobra.Command{
		Use:           "serve",
		Short:         "Run web server (with migrations)",
		SilenceErrors: true,
		RunE:          s.Run,
	}
}

func (s *serveCommand) Run(cmd *cobra.Command, args []string) error {
	s.log.Info("serve command started")
	waitCtx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-waitCtx.Done()
	s.log.Info("serve command stopped")
	return nil
}

type pingCommand struct {
	ping *pingService

	shutdowner fx.Shutdowner
}

func NewPingCommand(shutdowner fx.Shutdowner, ping *pingService) *pingCommand {
	return &pingCommand{ping: ping, shutdowner: shutdowner}
}

func (p *pingCommand) Command() *cobra.Command {
	c := &cobra.Command{
		Use:           "ping",
		Short:         "Run lightweight ping without migrations",
		SilenceErrors: true,
		RunE:          p.Run,
	}
	c.Flags().String("target", "local", "ping target label")
	return c
}

func (p *pingCommand) Run(cmd *cobra.Command, args []string) error {
	defer p.shutdowner.Shutdown()

	target, err := cmd.Flags().GetString("target")
	if err != nil {
		return err
	}
	return p.ping.Ping(cmd.Context(), target)
}

type testSubscriber struct {
}

func NewTestSubscriber() *testSubscriber {
	return &testSubscriber{}
}

func (t *testSubscriber) Subscribe(r mqtt.TopicRegistrar) error {
	var err error
	err = multierr.Append(err, r.Topic("hello", t.print))
	return err
}

func (t *testSubscriber) print(c realtime.Ctx) {
	fmt.Println("Received message:", c.Payload())
}

func main() {
	fx.New(
		di.App(
			di.Diagnostics(),
			otel.Module(),
			otel.UseRuntimeMetrics(),
			lifecycle.Module(),
			realtime.Module(
				realtime.UseAllowHook(),
				realtime.UseWebsocketListener(),
				realtime.UseTCPListener(),
			),
			database.Module(
				database.UseMigrations(&migrations),
			),
			web.Module(
				web.UseOtel(),
				web.UseSpa(web.WithSpaAssets(&assets)),
				web.UseSwagger(),
			),
			di.Provide(NewTestSubscriber),
			cmd.Module(
				cmd.UseBasicCommands(),
				di.Supply(&cobra.Command{
					Use: us.Name,
				}),
				cmd.Use("run",
					di.Provide(NewWorkerService),
					di.Provide(NewHandler, di.AsSelf[realtime.Authorizer]()),
					di.Provide(NewAPIService),
					database.UseOtel(),
					database.RunCheck(),
					database.RunMigrations(),
					web.RunFiberApp(),
				),
				cmd.Use("ping",
					di.Provide(NewPingService),
					di.Provide(NewPingCommand),
				),
			),
		).Build(),
	).Run()
}
