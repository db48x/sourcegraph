// Package router contains the URL router for the frontend app.
package router

import (
	"log"
	"net/url"
	"os"

	"github.com/sourcegraph/mux"
	"src.sourcegraph.com/sourcegraph/conf/feature"
	gitrouter "src.sourcegraph.com/sourcegraph/gitserver/router"
	"src.sourcegraph.com/sourcegraph/go-sourcegraph/routevar"
	"src.sourcegraph.com/sourcegraph/go-sourcegraph/spec"
)

const (
	BlogIndex     = "blog"
	BlogIndexAtom = "blog[.atom]"
	BlogPost      = "blog.post"

	Liveblog = "liveblog"

	Builds = "builds"

	Home = "home"

	RegisterClient = "register-client"

	DownloadInstall = "download.install"
	Download        = "download"

	RobotsTxt = "robots-txt"
	Favicon   = "favicon"

	SitemapIndex = "sitemap-index"
	RepoSitemap  = "repo.sitemap"

	User                      = "person"
	UserSettingsProfile       = "person.settings.profile"
	UserSettingsProfileAvatar = "person.settings.profile.avatar"
	UserSettingsKeys          = "person.settings.keys"

	Repo             = "repo"
	RepoBadge        = "repo.badge"
	RepoBadges       = "repo.badges"
	RepoCounter      = "repo.counter"
	RepoCounters     = "repo.counters"
	RepoCreate       = "repo.create"
	RepoBuilds       = "repo.builds"
	RepoBuild        = "repo.build"
	RepoBuildUpdate  = "repo.build.update"
	RepoBuildTaskLog = "repo.build.task.log"
	RepoBuildsCreate = "repo.builds.create"
	RepoSearch       = "repo.search"
	RepoRefresh      = "repo.refresh"
	RepoTree         = "repo.tree"
	RepoCompare      = "repo.compare"
	RepoCompareAll   = "repo.compare.all"

	RepoRevCommits = "repo.rev.commits"
	RepoCommit     = "repo.commit"
	RepoTags       = "repo.tags"
	RepoBranches   = "repo.branches"

	SearchForm    = "search.form"
	SearchResults = "search.results"

	LogIn          = "log-in"
	LogOut         = "log-out"
	SignUp         = "sign-up"
	ForgotPassword = "forgot-password"
	ResetPassword  = "reset-password"

	OAuth2ServerAuthorize = "oauth-provider.authorize"
	OAuth2ServerToken     = "oauth-provider.token"

	GitHubOAuth2Initiate = "github-oauth2.initiate"
	GitHubOAuth2Receive  = "github-oauth2.receive"

	Def         = "def"
	DefExamples = "def.examples"
	DefPopover  = "def.popover"

	UserContent = "usercontent"

	Markdown = "markdown"

	// Platform routes
	RepoAppFrame       = "repo.appframe"
	RepoPlatformSearch = "repo.platformsearch"

	// TODO: Cleanup.
	AppGlobalNotificationCenter = "appglobal.notifications"
)

// Router is an app URL router.
type Router struct{ mux.Router }

// New creates a new app router with route URL pattern definitions but
// no handlers attached to the routes.
//
// It is in a separate package from app so that other packages may use it to
// generate URLs without resulting in Go import cycles (and so we can release
// the router as open-source to support our client library).
func New(base *mux.Router) *Router {
	if base == nil {
		base = mux.NewRouter()
	}

	base.StrictSlash(true)

	base.Path("/").Methods("GET").Name(Home)
	base.Path("/register-client").Methods("GET", "POST").Name(RegisterClient)

	base.PathPrefix("/blog/live").Name(Liveblog)

	base.Path(`/blog`).Methods("GET").Name(BlogIndex)
	base.Path(`/blog{Format:\.atom}`).Methods("GET").Name(BlogIndexAtom)
	base.Path("/blog/{Slug:.*}").Methods("GET").Name(BlogPost)

	base.Path("/.builds").Methods("GET").Name(Builds)

	base.Path("/.download/install.sh").Methods("GET").Name(DownloadInstall)
	base.Path("/.download/{Suffix:.*}").Methods("GET").Name(Download)

	base.Path("/search").Methods("GET").Queries("q", "").Name(SearchResults)
	base.Path("/search").Methods("GET").Name(SearchForm)

	base.Path("/login").Methods("GET", "POST").Name(LogIn)
	base.Path("/join").Methods("GET", "POST").Name(SignUp)
	base.Path("/logout").Methods("POST").Name(LogOut)
	base.Path("/forgot").Methods("GET", "POST").Name(ForgotPassword)
	base.Path("/reset").Methods("GET", "POST").Name(ResetPassword)

	base.Path("/login/oauth/authorize").Methods("GET").Name(OAuth2ServerAuthorize)
	base.Path("/login/oauth/token").Methods("POST").Name(OAuth2ServerToken)

	base.Path("/robots.txt").Methods("GET").Name(RobotsTxt)
	base.Path("/favicon.ico").Methods("GET").Name(Favicon)

	base.Path("/sitemap.xml").Methods("GET").Name(SitemapIndex)

	base.Path("/github-oauth/initiate").Methods("GET").Name(GitHubOAuth2Initiate)
	base.Path("/github-oauth/receive").Methods("GET", "POST").Name(GitHubOAuth2Receive)

	base.Path("/usercontent/{Name}").Methods("GET").Name(UserContent)

	base.Path("/.markdown").Methods("POST").Name(Markdown)

	base.Path("/.create-repo").Methods("POST").Name(RepoCreate)

	// User routes begin with tilde (~).
	userPath := `/~` + routevar.User
	user := base.PathPrefix(userPath).Subrouter()
	user.Path("/.settings/profile").Methods("GET", "POST").Name(UserSettingsProfile)
	user.Path("/.settings/profile/avatar").Methods("POST").Name(UserSettingsProfileAvatar)
	user.Path("/.settings/keys").Methods("GET", "POST").Name(UserSettingsKeys)

	// attach git transport endpoints
	gitrouter.New(base)

	repo := base.PathPrefix(`/` + routevar.Repo).Subrouter()

	repoRevPath := `/` + routevar.RepoRev
	base.Path(repoRevPath).Methods("GET").PostMatchFunc(routevar.FixRepoRevVars).BuildVarsFunc(routevar.PrepareRepoRevRouteVars).Name(Repo)
	repoRev := base.PathPrefix(repoRevPath).PostMatchFunc(routevar.FixRepoRevVars).BuildVarsFunc(routevar.PrepareRepoRevRouteVars).Subrouter()

	// See router_util/def_route.go for an explanation of how we match def
	// routes.
	defPath := "/" + routevar.Def
	repoRev.Path(defPath).Methods("GET").PostMatchFunc(routevar.FixDefUnitVars).BuildVarsFunc(routevar.PrepareDefRouteVars).Name(Def)
	def := repoRev.PathPrefix(defPath).PostMatchFunc(routevar.FixDefUnitVars).BuildVarsFunc(routevar.PrepareDefRouteVars).Subrouter()
	def.Path("/.examples").Methods("GET").Name(DefExamples)
	def.Path("/.popover").Methods("GET").Name(DefPopover)
	// TODO(x): def history route

	// See router_util/tree_route.go for an explanation of how we match tree
	// entry routes.
	repoTreePath := "/.tree" + routevar.TreeEntryPath
	repoRev.Path(repoTreePath).Methods("GET").PostMatchFunc(routevar.FixTreeEntryVars).BuildVarsFunc(routevar.PrepareTreeEntryRouteVars).Name(RepoTree)

	repoRev.Path("/.refresh").Methods("POST", "PUT").Name(RepoRefresh)
	repoRev.Path("/.badges").Methods("GET").Name(RepoBadges)
	repoRev.Path("/.badges/{Badge}.{Format}").Methods("GET").Name(RepoBadge)
	repoRev.Path("/.search").Methods("GET").Name(RepoSearch)

	repoRev.Path("/.counters").Methods("GET").Name(RepoCounters)

	repoRev.Path("/.counters/{Counter}.{Format}").Methods("GET").Name(RepoCounter)
	repoRev.Path("/.commits").Methods("GET").Name(RepoRevCommits)

	headVar := "{Head:" + routevar.NamedToNonCapturingGroups(spec.RevPattern) + "}"
	repoRev.Path("/.compare/" + headVar).Methods("GET").Name(RepoCompare)
	repoRev.Path("/.compare/" + headVar + "/.all").Methods("GET").Name(RepoCompareAll)

	repo.Path("/.commits/{Rev:" + spec.PathNoLeadingDotComponentPattern + "}").Methods("GET").Name(RepoCommit)
	repo.Path("/.branches").Methods("GET").Name(RepoBranches)
	repo.Path("/.tags").Methods("GET").Name(RepoTags)
	repo.Path("/.sitemap.xml").Methods("GET").Name(RepoSitemap)

	repo.Path("/.builds").Methods("GET").Name(RepoBuilds)
	repo.Path("/.builds").Methods("POST").Name(RepoBuildsCreate)
	repoBuildPath := `/.builds/{Build:\d+}`
	repo.Path(repoBuildPath).Methods("GET").Name(RepoBuild)
	repo.Path(repoBuildPath).Methods("POST").Name(RepoBuildUpdate)
	repoBuild := repo.PathPrefix(repoBuildPath).Subrouter()
	repoBuild.Path(`/tasks/{Task:\d+}/log`).Methods("GET").Name(RepoBuildTaskLog)

	// This route dispatches to all SearchFrames that were registered through
	// RegisterSearchFrame in the platform package.
	repoRev.Path("/.search/{AppID}").Methods("GET").Name(RepoPlatformSearch)

	// This route should be AFTER all other repo/repoRev routes;
	// otherwise it will match every subroute.
	//
	// App is the app ID (e.g., "issues"), and AppPath is an opaque
	// path that Sourcegraph passes directly to the app. The empty
	// AppPath is the app's homepage, and it manages its own subpaths.
	repoRev.PathPrefix(`/.{App}{AppPath:(?:/.*)?}`).Name(RepoAppFrame)

	if feature.Features.NotificationCenter {
		// TODO.
		base.PathPrefix("/.notifications").Methods("GET").Name(AppGlobalNotificationCenter)
	}

	return &Router{*base}
}

func (r *Router) URLToOrError(routeName string, params ...string) (*url.URL, error) {
	route := r.Get(routeName)
	if route == nil {
		log.Panicf("no such route: %q (params: %v)", routeName, params)
	}
	u, err := route.URL(params...)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Router) URLTo(routeName string, params ...string) *url.URL {
	u, err := r.URLToOrError(routeName, params...)
	if err != nil {
		if os.Getenv("STRICT_URL_GEN") != "" && *u == (url.URL{}) {
			log.Panicf("Failed to generate route. See log message above.")
		}
		log.Printf("Route error: failed to make URL for route %q (params: %v): %s", routeName, params, err)
		return &url.URL{}
	}
	return u
}

var Rel = New(nil)
