package di

import (
	projectsv1 "LogFlux/contracts/gen/go/projects/v1"
	"LogFlux/services/projects/internal/config"
	"LogFlux/services/projects/internal/repository"
	"LogFlux/services/projects/internal/repository/postgres"
	"LogFlux/services/projects/internal/service"
	grpctransport "LogFlux/services/projects/internal/transport/grpc"
	httptransport "LogFlux/services/projects/internal/transport/http"
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Container struct {
	pool        *pgxpool.Pool
	httpHandler http.Handler
	grpcHandler projectsv1.ProjectAuthServiceServer
}

func NewContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	pool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		return nil, err
	}

	accessRepo := repository.NewAccessRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)

	auth, err := service.NewAuthService(userRepo, sessionRepo, cfg.Auth.SessionTTL)
	if err != nil {
		pool.Close()
		return nil, err
	}

	access := service.NewAccessService(accessRepo)
	apiKeys := service.NewAPIKeyValidator(
		repository.NewAPIKeyRepository(pool),
		access,
	)
	teams := service.NewTeamService(repository.NewTeamRepository(pool), accessRepo)
	projects := service.NewProjectService(repository.NewProjectRepository(pool), accessRepo)

	handler := httptransport.NewHandler(auth, teams, projects, apiKeys)

	return &Container{
		pool:        pool,
		httpHandler: httptransport.NewRouter(handler),
		grpcHandler: grpctransport.NewServer(apiKeys),
	}, nil
}
func (c *Container) HTTPHandler() http.Handler {
	return c.httpHandler
}

func (c *Container) GRPCHandler() projectsv1.ProjectAuthServiceServer {
	return c.grpcHandler
}

func (c *Container) Close() error {
	c.pool.Close()
	return nil
}
