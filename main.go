package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GenerateRequest struct {
	Prompt   string `json:"prompt" binding:"required,max=500"`
	ParentID int    `json:"parent_id"`
}

func resolveParent(store *Store, requestedParentID int) (int, string, error) {
	if requestedParentID == 0 {
		if latest, ok := store.Latest(); ok {
			return latest.ID, latest.HTML, nil
		}
		return 0, "", nil
	}

	parent, ok := store.Get(requestedParentID)
	if !ok {
		return 0, "", errors.New("selected version not found")
	}
	return parent.ID, parent.HTML, nil
}

// serveResources serves logos and fonts. The generated page runs in a
// sandboxed iframe with origin "null", and browsers only load cross-origin
// fonts when CORS allows it, so these public files allow any origin.
func serveResources(r *gin.Engine) {
	r.Group("/resources", func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
	}).Static("/", "./resources")
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	// gin.Default initializes a new Gin router with default middleware (logger and recovery)
	r := gin.Default()
	store, err := NewStore("sections.json")
	if err != nil {
		log.Fatal("could not open store: ", err)
	}

	serveResources(r)

	r.GET("/sections", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"sections": store.All()})
	})
	// Load HTML Templates into the Gin router.
	r.LoadHTMLGlob("templates/*")

	// r.GET defines a route for the HTTP GET method at the root path ("/").
	// When this route is accessed, it executes the provided handler function,
	// in this case, rendering the "index.html" template with a title variable.
	r.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	r.POST("/generate", func(ctx *gin.Context) {
		var req GenerateRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		parentID, previous, err := resolveParent(store, req.ParentID)
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Selected version not found"})
			return
		}

		if ok, reason := requestAllowed(ctx.Request.Context(), req.Prompt); !ok {
			log.Println("REQUEST REJECTED:", reason)
			msg := "This request was not accepted. Try phrasing it another way."
			if reason == reasonExternal {
				msg = "The page can't have external links or embedded videos. Try building it into the page itself."
			}
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": msg})
			return
		}

		html, err := generateChecked(ctx.Request.Context(), req.Prompt, previous)
		if err != nil {
			log.Println("GENERATE ERROR:", err)
			msg := "FIM can't respond right now, try again"
			if errors.Is(err, errQuality) {
				msg = "FIM wasn't happy with the result (quality check). Tweak your prompt a little and try again."
			}
			ctx.JSON(http.StatusBadGateway, gin.H{"error": msg})
			return
		}

		sec, err := store.Add(parentID, req.Prompt, html)
		if err != nil {
			log.Println("STORE ERROR:", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save the section"})
			return
		}

		ctx.JSON(http.StatusOK, sec)
	})

	// r.Static serves static files from the specified directory.
	r.Static("/public", "./public")

	// r.Run() starts the HTTP server on the defined address (default is ":8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("server stopped: ", err)
	}
}
