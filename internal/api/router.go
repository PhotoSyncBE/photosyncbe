package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(authHandler *AuthHandler, photoHandler *PhotoHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Post("/api/auth/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(JWTMiddleware(authHandler.jwtManager))

		r.Get("/api/photos", photoHandler.ListPhotos)
		r.Post("/api/photos", photoHandler.UploadPhoto)
		r.Get("/api/photos/{id}", photoHandler.DownloadPhoto)
		r.Get("/api/photos/{id}/thumbnail", photoHandler.GetThumbnail)
		r.Delete("/api/photos/{id}", photoHandler.DeletePhoto)
		r.Get("/api/photos/{id}/info", photoHandler.GetPhotoInfo)
	})

	return r
}
