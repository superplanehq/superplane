package public

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	grpcLib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// Event payload can be up to 64k in size
	MaxEventSize = 64 * 1024

	// The size of a stage execution can be up to 4k
	MaxExecutionTagsSize = 4 * 1024
)

type Server struct {
	httpServer            *http.Server
	encryptor             encryptor.Encryptor
	jwt                   *jwt.Signer
	timeoutHandlerTimeout time.Duration
	Router                *mux.Router
	BasePath              string
}

func NewServer(encryptor encryptor.Encryptor, jwtSigner *jwt.Signer, basePath string, middlewares ...mux.MiddlewareFunc) (*Server, error) {
	server := &Server{
		timeoutHandlerTimeout: 15 * time.Second,
		encryptor:             encryptor,
		jwt:                   jwtSigner,
		BasePath:              basePath,
	}

	server.timeoutHandlerTimeout = 15 * time.Second
	server.InitRouter(middlewares...)
	return server, nil
}

func (s *Server) RegisterGRPCGateway(grpcServerAddr string) error {
	ctx := context.Background()

	grpcGatewayMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			return key, true
		}),
	)

	opts := []grpcLib.DialOption{grpcLib.WithTransportCredentials(insecure.NewCredentials())}

	err := pb.RegisterSuperplaneHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = grpcGatewayMux.HandlePath("GET", "/api/v1/canvases/is-alive", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.WriteHeader(http.StatusOK)
	})

	if err != nil {
		return err
	}

	// This is necessary because it is not possible to use the current router with
	// runtime Mux. Runtime mux has no specification for the added paths and it the only
	// supported tool for grpc-gateway.
	s.Router.PathPrefix("/api/v1/canvases").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		grpcGatewayMux.ServeHTTP(w, r2)
	}))

	return nil
}

// RegisterOpenAPIHandler adds handlers to serve the OpenAPI specification and Swagger UI
func (s *Server) RegisterOpenAPIHandler() {
	swaggerFilesPath := os.Getenv("SWAGGER_BASE_PATH")
	if swaggerFilesPath == "" {
		log.Errorf("SWAGGER_BASE_PATH is not set")
		return
	}

	if _, err := os.Stat(swaggerFilesPath); os.IsNotExist(err) {
		log.Errorf("API documentation directory %s does not exist", swaggerFilesPath)
		return
	}

	s.Router.HandleFunc(s.BasePath+"/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, swaggerFilesPath+"/swagger-ui.html")
	})

	s.Router.HandleFunc(s.BasePath+"/docs/superplane.swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, swaggerFilesPath+"/superplane.swagger.json")
	})

	log.Infof("OpenAPI specification available at %s", swaggerFilesPath)
	log.Infof("Swagger UI available at %s", swaggerFilesPath)
	log.Infof("Raw API JSON available at %s", swaggerFilesPath+"/superplane.swagger.json")
}

func (s *Server) InitRouter(additionalMiddlewares ...mux.MiddlewareFunc) {
	r := mux.NewRouter().StrictSlash(true)

	//
	// Authenticated and validated routes.
	//
	authenticatedRoute := r.Methods(http.MethodPost).Subrouter()

	authenticatedRoute.
		HandleFunc(s.BasePath+"/sources/{sourceID}/github", s.HandleGithubWebhook).
		Headers("Content-Type", "application/json").
		Methods("POST")

	authenticatedRoute.
		HandleFunc(s.BasePath+"/sources/{sourceID}/semaphore", s.HandleSemaphoreWebhook).
		Headers("Content-Type", "application/json").
		Methods("POST")

	authenticatedRoute.
		HandleFunc(s.BasePath+"/executions/{executionID}/tags", s.HandleExecutionTags).
		Headers("Content-Type", "application/json").
		Methods("POST")

	authenticatedRoute.Use(additionalMiddlewares...)

	//
	// No authentication of any kind, just a health endpoint
	//
	unauthenticatedRoute := r.Methods(http.MethodGet).Subrouter()
	unauthenticatedRoute.HandleFunc("/", s.HealthCheck).Methods("GET")

	s.Router = r
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) Serve(host string, port int) error {
	log.Infof("Starting server at %s:%d", host, port)
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler: http.TimeoutHandler(
			handlers.LoggingHandler(os.Stdout, s.Router),
			s.timeoutHandlerTimeout,
			"request timed out",
		),
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Close() {
	if err := s.httpServer.Close(); err != nil {
		log.Errorf("Error closing server: %v", err)
	}
}

func (s *Server) HandleExecutionTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID, err := uuid.Parse(vars["executionID"])
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	execution, err := models.FindExecutionByID(executionID)
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	headerParts := strings.Split(authHeader, "Bearer ")
	if len(headerParts) != 2 {
		http.Error(w, "Malformed Authorization header", http.StatusUnauthorized)
		return
	}

	token := headerParts[1]
	err = s.jwt.Validate(token, execution.ID.String())
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxExecutionTagsSize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(
				w,
				fmt.Sprintf("Request body is too large - must be up to %d bytes", MaxExecutionTagsSize),
				http.StatusRequestEntityTooLarge,
			)

			return
		}

		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	err = execution.AddTags(body)
	if err != nil {
		http.Error(w, "Error updating tags", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleGithubWebhook(w http.ResponseWriter, r *http.Request) {
	//
	// Any verification that happens here must be quick
	// so we always respond with a 200 OK to the event origin.
	// All the event processing happen on the workers.
	//

	vars := mux.Vars(r)
	sourceIDFromRequest := vars["sourceID"]
	sourceID, err := uuid.Parse(sourceIDFromRequest)
	if err != nil {
		http.Error(w, "source ID not found", http.StatusNotFound)
		return
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "Missing X-Hub-Signature-256 header", http.StatusBadRequest)
		return
	}

	// TODO: we don't have the canvas ID here.
	// We could put it in the path, but then the path will become quite big.
	// For now, just organization/source IDs are enough for us.
	source, err := models.FindEventSource(sourceID)
	if err != nil {
		http.Error(w, "source ID not found", http.StatusNotFound)
		return
	}

	//
	// Only read up to the maximum event size we allow,
	// and only proceed if payload is below that.
	//
	r.Body = http.MaxBytesReader(w, r.Body, MaxEventSize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(
				w,
				fmt.Sprintf("Request body is too large - must be up to %d bytes", MaxEventSize),
				http.StatusRequestEntityTooLarge,
			)

			return
		}

		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	headers, err := parseHeaders(&r.Header)
	if err != nil {
		http.Error(w, "Error parsing headers", http.StatusBadRequest)
		return
	}

	key, err := s.encryptor.Decrypt(r.Context(), source.Key, []byte(source.Name))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	signature = strings.Replace(signature, "sha256=", "", 1)
	if err := crypto.VerifySignature(key, body, signature); err != nil {
		log.Errorf("Invalid signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	//
	// Here, we know the event is for a valid organization/source,
	// and comes from GitHub, so we just want to save it and give a response back.
	//
	if _, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, body, headers); err != nil {
		http.Error(w, "Error receiving event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleSemaphoreWebhook(w http.ResponseWriter, r *http.Request) {
	//
	// Any verification that happens here must be quick
	// so we always respond with a 200 OK to the event origin.
	// All the event processing happen on the workers.
	//

	vars := mux.Vars(r)
	sourceIDFromRequest := vars["sourceID"]
	sourceID, err := uuid.Parse(sourceIDFromRequest)
	if err != nil {
		http.Error(w, "source ID not found", http.StatusNotFound)
		return
	}

	signature := r.Header.Get("X-Semaphore-Signature-256")
	if signature == "" {
		http.Error(w, "Missing X-Semaphore-Signature-256 header", http.StatusBadRequest)
		return
	}

	source, err := models.FindEventSource(sourceID)
	if err != nil {
		http.Error(w, "source ID not found", http.StatusNotFound)
		return
	}

	//
	// Only read up to the maximum event size we allow,
	// and only proceed if payload is below that.
	//
	r.Body = http.MaxBytesReader(w, r.Body, MaxEventSize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(
				w,
				fmt.Sprintf("Request body is too large - must be up to %d bytes", MaxEventSize),
				http.StatusRequestEntityTooLarge,
			)

			return
		}

		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	headers, err := parseHeaders(&r.Header)
	if err != nil {
		http.Error(w, "Error parsing headers", http.StatusBadRequest)
		return
	}

	key, err := s.encryptor.Decrypt(r.Context(), source.Key, []byte(source.Name))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	signature = strings.Replace(signature, "sha256=", "", 1)

	if err := crypto.VerifySignature(key, body, signature); err != nil {
		log.Errorf("Invalid signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	//
	// Here, we know the event is for a valid organization/source,
	// and comes from Semaphore, so we just want to save it and give a response back.
	//
	if _, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, body, headers); err != nil {
		http.Error(w, "Error receiving event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func parseHeaders(headers *http.Header) ([]byte, error) {
	parsedHeaders := make(map[string]string, len(*headers))
	for key, value := range *headers {
		parsedHeaders[key] = value[0]
	}

	return json.Marshal(parsedHeaders)
}
