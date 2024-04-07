package save

import (
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"
)

type Request struct {
	URL   string `json:"url" validate:"required"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	response.Response
	Alias string `json:"alias,omitempty"`
}

type URLSaver interface {
	SaveURL(urlToSave string, alias string) error
}

const aliasLength = 8

func New(log *slog.Logger, urlSaver URLSaver, domain string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log = log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req Request

		err := render.DecodeJSON(r.Body, &req)

		if err != nil {
			log.Error("failed to decode request body", sl.Error(err))

			render.JSON(w, r, response.Error("failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Error(err))

			render.JSON(w, r, response.ValidationError(validateErr))

			return
		}

		alias := req.Alias
		if alias == "" {
			alias, err = random.NewRandomString(aliasLength)
			if err != nil {
				log.Error("internal server error", sl.Error(err))

				render.JSON(w, r, response.Error("internal server error"))

				return
			}
		}

		err = urlSaver.SaveURL(req.URL, alias)

		if errors.Is(err, storage.ErrURLExists) {
			log.Info("url already exists", slog.String("url", req.URL))

			render.JSON(w, r, response.Error("url already exists"))

			return
		}

		render.JSON(w, r, Response{
			Response: response.OK(),
			Alias:    domain + alias,
		})

		return
	}
}
