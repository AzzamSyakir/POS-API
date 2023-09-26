package responses

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse mengembalikan respons JSON berupa pesan kesalahan.
func ErrorResponse(w http.ResponseWriter, message string, status int) {
	type Response struct {
		Error   bool   `json:"Error"`
		Message string `json:"message"`
	}

	response := Response{
		Error:   false,
		Message: message,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		ErrorResponse(w, "Gagal membuat respons JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(responseJSON)
}

func SuccessResponse(w http.ResponseWriter, message string, data interface{}, status int) {
	type Response struct {
		Success bool        `json:"success"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}

	response := Response{
		Success: true,
		Message: message,
		Data:    data,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		ErrorResponse(w, "Gagal membuat respons JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(responseJSON)
}
func OtherResponses(w http.ResponseWriter, message string, status int) {
	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	response := Response{
		Success: true,
		Message: message,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		ErrorResponse(w, "Gagal membuat respons JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(responseJSON)
}
