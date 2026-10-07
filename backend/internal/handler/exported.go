package handler

import (
	"net/http"
)

// Exported handler methods for the router. Each simply delegates to the
// unexported implementation so handler logic stays in one file.

func (a *API) HandleHealthz(w http.ResponseWriter, r *http.Request)   { a.handleHealthz(w, r) }
func (a *API) HandleReadyz(w http.ResponseWriter, r *http.Request)    { a.handleReadyz(w, r) }
func (a *API) HandleCreateURL(w http.ResponseWriter, r *http.Request) { a.handleCreateURL(w, r) }
func (a *API) HandleListURLs(w http.ResponseWriter, r *http.Request)  { a.handleListURLs(w, r) }
func (a *API) HandleGetURL(w http.ResponseWriter, r *http.Request)    { a.handleGetURL(w, r) }
func (a *API) HandlePatchURL(w http.ResponseWriter, r *http.Request)  { a.handlePatchURL(w, r) }
func (a *API) HandleDeleteURL(w http.ResponseWriter, r *http.Request) { a.handleDeleteURL(w, r) }
func (a *API) HandleStats(w http.ResponseWriter, r *http.Request)     { a.handleStats(w, r) }
func (a *API) HandleOverview(w http.ResponseWriter, r *http.Request)  { a.handleOverview(w, r) }
func (a *API) HandleRegister(w http.ResponseWriter, r *http.Request)  { a.handleRegister(w, r) }
func (a *API) HandleLogin(w http.ResponseWriter, r *http.Request)     { a.handleLogin(w, r) }
func (a *API) HandleLogout(w http.ResponseWriter, r *http.Request)    { a.handleLogout(w, r) }
func (a *API) HandleMe(w http.ResponseWriter, r *http.Request)        { a.handleMe(w, r) }
func (a *API) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	a.handleChangePassword(w, r)
}

// HandleRedirect serves GET /{code}.
func (a *API) HandleRedirect(w http.ResponseWriter, r *http.Request) { a.handleRedirect(w, r) }

// HandleRedirectNoClick serves HEAD /{code} without counting a click.
func (a *API) HandleRedirectNoClick(w http.ResponseWriter, r *http.Request) {
	a.handleRedirectNoClick(w, r)
}

// HandleNotFound renders the final 404 page.
func (a *API) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	a.serveErrorPage(w, r, http.StatusNotFound)
}
