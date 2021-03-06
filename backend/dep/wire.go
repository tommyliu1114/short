//+build wireinject

package dep

import (
	"database/sql"

	"github.com/google/wire"
	"github.com/short-d/app/fw/analytics"
	"github.com/short-d/app/fw/cli"
	"github.com/short-d/app/fw/db"
	"github.com/short-d/app/fw/env"
	"github.com/short-d/app/fw/geo"
	"github.com/short-d/app/fw/graphql"
	"github.com/short-d/app/fw/io"
	"github.com/short-d/app/fw/logger"
	"github.com/short-d/app/fw/metrics"
	"github.com/short-d/app/fw/network"
	"github.com/short-d/app/fw/runtime"
	"github.com/short-d/app/fw/service"
	"github.com/short-d/app/fw/timer"
	"github.com/short-d/app/fw/webreq"
	"github.com/short-d/short/backend/app/adapter/facebook"
	"github.com/short-d/short/backend/app/adapter/github"
	"github.com/short-d/short/backend/app/adapter/google"
	"github.com/short-d/short/backend/app/adapter/gqlapi"
	"github.com/short-d/short/backend/app/adapter/kgs"
	"github.com/short-d/short/backend/app/adapter/request"
	"github.com/short-d/short/backend/app/adapter/sqldb"
	"github.com/short-d/short/backend/app/usecase/account"
	"github.com/short-d/short/backend/app/usecase/changelog"
	"github.com/short-d/short/backend/app/usecase/external"
	"github.com/short-d/short/backend/app/usecase/repository"
	"github.com/short-d/short/backend/app/usecase/requester"
	"github.com/short-d/short/backend/app/usecase/risk"
	"github.com/short-d/short/backend/app/usecase/url"
	"github.com/short-d/short/backend/app/usecase/validator"
	"github.com/short-d/short/backend/dep/provider"
)

var authSet = wire.NewSet(
	provider.NewJwtGo,
	provider.NewAuthenticator,
)

var observabilitySet = wire.NewSet(
	wire.Bind(new(io.Output), new(io.StdOut)),
	wire.Bind(new(runtime.Runtime), new(runtime.Program)),
	wire.Bind(new(metrics.Metrics), new(metrics.DataDog)),
	wire.Bind(new(analytics.Analytics), new(analytics.Segment)),
	wire.Bind(new(network.Network), new(network.Proxy)),

	io.NewStdOut,
	provider.NewEntryRepositorySwitch,
	provider.NewLogger,
	runtime.NewProgram,
	provider.NewDataDogMetrics,
	provider.NewSegment,
	network.NewProxy,
	request.NewClient,
	request.NewInstrumentationFactory,
)

var githubAPISet = wire.NewSet(
	provider.NewGithubIdentityProvider,
	github.NewAccount,
	github.NewAPI,
)

var facebookAPISet = wire.NewSet(
	provider.NewFacebookIdentityProvider,
	facebook.NewAccount,
	facebook.NewAPI,
)

var googleAPISet = wire.NewSet(
	provider.NewGoogleIdentityProvider,
	google.NewAccount,
	google.NewAPI,
)

var keyGenSet = wire.NewSet(
	wire.Bind(new(external.KeyFetcher), new(kgs.RPC)),
	provider.NewKgsRPC,
	provider.NewKeyGenerator,
)

var featureDecisionSet = wire.NewSet(
	wire.Bind(new(repository.FeatureToggle), new(sqldb.FeatureToggleSQL)),
	sqldb.NewFeatureToggleSQL,
	provider.NewFeatureDecisionMakerFactorySwitch,
)

// InjectCommandFactory creates CommandFactory with configured dependencies.
func InjectCommandFactory() cli.CommandFactory {
	wire.Build(
		wire.Bind(new(cli.CommandFactory), new(cli.CobraFactory)),
		cli.NewCobraFactory,
	)
	return cli.CobraFactory{}
}

// InjectDBConnector creates DBConnector with configured dependencies.
func InjectDBConnector() db.Connector {
	wire.Build(
		wire.Bind(new(db.Connector), new(db.PostgresConnector)),
		db.NewPostgresConnector,
	)
	return db.PostgresConnector{}
}

// InjectDBMigrationTool creates DBMigrationTool with configured dependencies.
func InjectDBMigrationTool() db.MigrationTool {
	wire.Build(
		wire.Bind(new(db.MigrationTool), new(db.PostgresMigrationTool)),
		db.NewPostgresMigrationTool,
	)
	return db.PostgresMigrationTool{}
}

// InjectEnv creates Environment with configured dependencies.
func InjectEnv() env.Env {
	wire.Build(
		wire.Bind(new(env.Env), new(env.GoDotEnv)),
		env.NewGoDotEnv,
	)
	return env.GoDotEnv{}
}

// InjectGraphQLService creates GraphQL service with configured dependencies.
func InjectGraphQLService(
	runtime env.Runtime,
	prefix provider.LogPrefix,
	logLevel logger.LogLevel,
	sqlDB *sql.DB,
	graphqlPath provider.GraphQLPath,
	secret provider.ReCaptchaSecret,
	jwtSecret provider.JwtSecret,
	bufferSize provider.KeyGenBufferSize,
	kgsRPCConfig provider.KgsRPCConfig,
	tokenValidDuration provider.TokenValidDuration,
	dataDogAPIKey provider.DataDogAPIKey,
	segmentAPIKey provider.SegmentAPIKey,
	ipStackAPIKey provider.IPStackAPIKey,
	googleAPIKey provider.GoogleAPIKey,
) (service.GraphQL, error) {
	wire.Build(
		wire.Bind(new(timer.Timer), new(timer.System)),
		wire.Bind(new(graphql.API), new(gqlapi.Short)),
		wire.Bind(new(graphql.Handler), new(graphql.GraphGopherHandler)),

		wire.Bind(new(risk.BlackList), new(google.SafeBrowsing)),
		wire.Bind(new(repository.UserURLRelation), new(sqldb.UserURLRelationSQL)),
		wire.Bind(new(repository.ChangeLog), new(sqldb.ChangeLogSQL)),
		wire.Bind(new(repository.URL), new(*sqldb.URLSql)),

		wire.Bind(new(changelog.ChangeLog), new(changelog.Persist)),
		wire.Bind(new(url.Retriever), new(url.RetrieverPersist)),
		wire.Bind(new(url.Creator), new(url.CreatorPersist)),

		observabilitySet,
		authSet,
		keyGenSet,

		env.NewDeployment,
		provider.NewGraphQLService,
		graphql.NewGraphGopherHandler,
		webreq.NewHTTPClient,
		webreq.NewHTTP,
		timer.NewSystem,

		gqlapi.NewShort,
		provider.NewSafeBrowsing,
		risk.NewDetector,
		provider.NewReCaptchaService,
		sqldb.NewChangeLogSQL,
		sqldb.NewURLSql,
		sqldb.NewUserURLRelationSQL,

		validator.NewLongLink,
		validator.NewCustomAlias,
		changelog.NewPersist,
		url.NewRetrieverPersist,
		url.NewCreatorPersist,
		requester.NewVerifier,
	)
	return service.GraphQL{}, nil
}

// InjectRoutingService creates routing service with configured dependencies.
func InjectRoutingService(
	runtime env.Runtime,
	prefix provider.LogPrefix,
	logLevel logger.LogLevel,
	sqlDB *sql.DB,
	githubClientID provider.GithubClientID,
	githubClientSecret provider.GithubClientSecret,
	facebookClientID provider.FacebookClientID,
	facebookClientSecret provider.FacebookClientSecret,
	facebookRedirectURI provider.FacebookRedirectURI,
	googleClientID provider.GoogleClientID,
	googleClientSecret provider.GoogleClientSecret,
	googleRedirectURI provider.GoogleRedirectURI,
	jwtSecret provider.JwtSecret,
	bufferSize provider.KeyGenBufferSize,
	kgsRPCConfig provider.KgsRPCConfig,
	webFrontendURL provider.WebFrontendURL,
	tokenValidDuration provider.TokenValidDuration,
	dataDogAPIKey provider.DataDogAPIKey,
	segmentAPIKey provider.SegmentAPIKey,
	ipStackAPIKey provider.IPStackAPIKey,
) (service.Routing, error) {
	wire.Build(
		wire.Bind(new(timer.Timer), new(timer.System)),
		wire.Bind(new(geo.Geo), new(geo.IPStack)),

		wire.Bind(new(url.Retriever), new(url.RetrieverPersist)),
		wire.Bind(new(repository.UserURLRelation), new(sqldb.UserURLRelationSQL)),
		wire.Bind(new(repository.User), new(*sqldb.UserSQL)),
		wire.Bind(new(repository.URL), new(*sqldb.URLSql)),

		observabilitySet,
		authSet,
		githubAPISet,
		facebookAPISet,
		googleAPISet,
		keyGenSet,
		featureDecisionSet,

		service.NewRouting,
		webreq.NewHTTPClient,
		webreq.NewHTTP,
		graphql.NewClientFactory,
		timer.NewSystem,
		provider.NewIPStack,
		env.NewDeployment,

		sqldb.NewUserSQL,
		sqldb.NewURLSql,
		sqldb.NewUserURLRelationSQL,
		url.NewRetrieverPersist,
		account.NewProvider,
		provider.NewShortRoutes,
	)
	return service.Routing{}, nil
}
