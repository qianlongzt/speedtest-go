package results

import (
	_ "embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/librespeed/speedtest/config"
	"github.com/librespeed/speedtest/database"
	"github.com/librespeed/speedtest/database/schema"
)

type StatsData struct {
	NoPassword bool
	LoggedIn   bool
	Data       []schema.TelemetryData
}

var (
	key   = []byte(securecookie.GenerateRandomKey(32))
	store = sessions.NewCookieStore(key)
	conf  *config.Config
)

func Initialize(c *config.Config) {
	conf = c
	store.Options = &sessions.Options{
		Path:     conf.BaseURL + "/stats",
		MaxAge:   3600 * 1, // 1 hour
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

func Stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if conf.DatabaseType == "none" {
		render.PlainText(w, r, "Statistics are disabled")
		return
	}

	var data StatsData

	if conf.StatsPassword == "PASSWORD" {
		data.NoPassword = true
	}

	if !data.NoPassword {
		op := r.FormValue("op")
		session, _ := store.Get(r, "logged")
		auth, ok := session.Values["authenticated"].(bool)

		if auth && ok {
			if op == "logout" {
				session.Values["authenticated"] = false
				session.Options.MaxAge = -1
				session.Save(r, w)
				http.Redirect(w, r, conf.BaseURL+"/stats", http.StatusTemporaryRedirect)
			} else {
				data.LoggedIn = true

				id := r.FormValue("id")
				switch id {
				case "L100":
					stats, err := database.DB.FetchLast100()
					if err != nil {
						slog.Error("fetching data from database", slog.Any("error", err))
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					data.Data = stats
				case "":
				default:
					stat, err := database.DB.FetchByUUID(id)
					if err != nil {
						slog.Error("fetching data from database", slog.Any("error", err))
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					data.Data = append(data.Data, *stat)
				}
			}
		} else {
			if op == "login" {
				session, _ := store.Get(r, "logged")
				password := r.FormValue("password")
				if password == conf.StatsPassword {
					session.Values["authenticated"] = true
					session.Save(r, w)
					http.Redirect(w, r, conf.BaseURL+"/stats", http.StatusTemporaryRedirect)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
			}
		}
	}

	if err := t.Execute(w, data); err != nil {
		slog.Error("executing template", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

var t *template.Template

func init() {
	var err error
	t, err = template.New("template").Parse(htmlTemplate)
	if err != nil {
		panic(fmt.Errorf("failed to parse template: %w", err))
	}
}

//go:embed stats.gohtml
var htmlTemplate string
