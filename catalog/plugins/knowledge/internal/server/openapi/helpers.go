package openapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

// Response returns an ImplResponse struct filled with the given code and body.
func Response(code int, body interface{}) ImplResponse {
	return ImplResponse{
		Code: code,
		Body: body,
	}
}

// EncodeJSONResponse uses the json encoder to write an interface to the http response with an optional status code.
func EncodeJSONResponse(i interface{}, status *int, w http.ResponseWriter) error {
	wHeader := w.Header()

	f, ok := i.(*os.File)
	if ok {
		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		wHeader.Set("Content-Type", http.DetectContentType(data))
		wHeader.Set("Content-Disposition", "attachment; filename="+f.Name())
		if status != nil {
			w.WriteHeader(*status)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_, err = w.Write(data)
		return err
	}
	wHeader.Set("Content-Type", "application/json; charset=UTF-8")

	if status != nil {
		w.WriteHeader(*status)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if i != nil {
		return json.NewEncoder(w).Encode(i)
	}

	return nil
}

// parseQuery parses query parameters and returns an error if any malformed value pairs are encountered.
func parseQuery(rawQuery string) (url.Values, error) {
	return url.ParseQuery(rawQuery)
}

// parseInt32 parses a string parameter to an int32.
func parseInt32(param string) (int32, error) {
	if param == "" {
		return 0, nil
	}
	val, err := strconv.ParseInt(param, 10, 32)
	return int32(val), err
}
