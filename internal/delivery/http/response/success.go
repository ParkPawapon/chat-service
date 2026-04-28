package response

import (
	"encoding/json"
	"net/http"
)

const ISOTimeLayout = "2006-01-02T15:04:05.000Z"

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
