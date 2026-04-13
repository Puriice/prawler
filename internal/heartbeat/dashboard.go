package heartbeat

import (
	"html/template"
	"net/http"
)

func (h *Holter) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.ParseFiles("./web/dashboard.html"))
	tmpl.Execute(w, nil)
}
