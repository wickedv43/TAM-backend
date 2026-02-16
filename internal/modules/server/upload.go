package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
)

// Upload uploads a file
// @Summary      Upload file
// @Description  Upload a file to the server
// @Tags         upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "File to upload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  string "Bad Request"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /upload [post]
func (s *Server) Upload(c echo.Context) error {
	// Source
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Destination
	// Create uploads directory if not exists
	uploadDir := "uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dstPath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// Construct URL
	// Assuming server is running on localhost or configured domain
	// Ideally, this should come from config
	serverURL := fmt.Sprintf("http://localhost:%s", s.cfg.Server.Port)

	fileURL := fmt.Sprintf("%s/uploads/%s", serverURL, filename)

	return c.JSON(http.StatusOK, map[string]string{
		"url": fileURL,
	})
}
