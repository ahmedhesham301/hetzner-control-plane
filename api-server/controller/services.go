package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/ahmedhesham301/hetzner-control-plane/api-server/data"
	"github.com/ahmedhesham301/hetzner-control-plane/api-server/temporal"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
)

func CreateService(g *gin.Context) {
	serviceValue, exists := g.Get("params")
	if !exists {
		g.Status(http.StatusInternalServerError)
		slog.Error("Required params missing from Gin context", "key", "params")
		return
	}
	service, ok := serviceValue.(data.Service)
	if !ok {
		g.Status(http.StatusInternalServerError)
		slog.Error(
			"Invalid params type in Gin context", "key", "params", "expected_type", "data.CreateServiceParams",
		)
		return
	}

	err, recordID := service.CreateRecord(context.Background())
	if err != nil {
		g.Status(http.StatusInternalServerError)
		slog.Error("Failed to create database record", "error", err)
		return
	}
	serviceJSON, err := json.Marshal(service)
	if err != nil {
		slog.Error("Failed to serialize service", "error", err)
		g.Status(http.StatusInternalServerError)
		return
	}

	options := client.StartWorkflowOptions{
		ID:        "create-Service-workflow" + strconv.Itoa(*recordID),
		TaskQueue: "task-queue",
	}

	networkvar := os.Getenv("networkID")
	networkID, err := strconv.ParseInt(networkvar, 10, 64)
	if err != nil {
		g.Status(http.StatusInternalServerError)
		slog.Error("Failed to Parse network id", "error", err)
		return
	}

	workflowParams := temporal.CreateServiceWorkflowParams{
		RawService:    serviceJSON,
		Env:           os.Getenv("ENV"),
		TemplatesPath: os.Getenv("PACKER_TEMPLATES_PATH"),
		NetworkID:     networkID,
	}
	_, err = temporal.TemporalClient.ExecuteWorkflow(context.TODO(), options, temporal.CreateServiceWorkflow, workflowParams)
	if err != nil {
		g.Status(http.StatusInternalServerError)
		slog.Error("Failed to Execute workflow", "error", err)
		return
	}
	g.JSON(http.StatusAccepted, service)
}

func ListMangedServices(g *gin.Context) {
	templatesPath := os.Getenv("PACKER_TEMPLATES_PATH")

	files, err := os.ReadDir(templatesPath)
	if err != nil {
		slog.Error("error", "failed to read so dir templates path", err)
		g.Status(http.StatusInternalServerError)
		return
	}
	services := []string{}
	for _, file := range files {
		if file.Name()[0] != '.' {
			services = append(services, file.Name())
		}
	}
	g.JSON(http.StatusOK, services)
}

func GetMangedServiceCreateOps(g *gin.Context) {
	serviceName := g.Param("kind")

	templatesPath := os.Getenv("PACKER_TEMPLATES_PATH")

	content, err := os.ReadFile(templatesPath + "/" + serviceName + "/info.json")
	if err != nil {
		slog.Error("Error when opening file", "err", err)
		g.JSON(http.StatusBadRequest, gin.H{"error": "no template called " + serviceName})
		return
	}

	var serviceInfo data.ServiceInfo
	err = json.Unmarshal(content, &serviceInfo)
	if err != nil {
		slog.Error("Error during Unmarshal", "err", err)
		return
	}
	g.JSON(http.StatusOK, serviceInfo)

}
