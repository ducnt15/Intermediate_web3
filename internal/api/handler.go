package api

import (
	"Intermediate_web3/internal/models"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Response struct {
	Status  string      `json:"Status"`
	Message string      `json:"Message"`
	Data    interface{} `json:"Data,omitempty"`
}

const (
	defaultPage     = 1
	defaultPageSize = 10
)

var (
	tracking []models.TrackingInformation
)

func SaveDB(trackingInfo *models.TrackingInformation) error {
	_, err := GetDB().NewInsert().Model(trackingInfo).Exec(context.Background())
	if err != nil {
		fmt.Println("Error inserting data into database:", err)
	}
	return nil
}

func GetTracking(c *gin.Context) {
	// Check for database connection
	if GetDB() == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "false",
			Message: "Database connection is not initialized",
		})
		return
	}
	ctx := context.Background()

	page, pageSize := getPageAndSize(c, defaultPage, defaultPageSize)

	totalRecords, err := GetDB().NewSelect().
		Model((*models.TrackingInformation)(nil)).
		Count(ctx)

	totalPages := (totalRecords + pageSize - 1) / pageSize

	tracking, err = GetPaginatedTracking(ctx, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "false",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "true",
		Message: "Get all tracking successfully!",
		Data: struct {
			TotalPages int                          `json:"totalPages"`
			Tracking   []models.TrackingInformation `json:"tracking"`
		}{
			TotalPages: totalPages,
			Tracking:   tracking,
		},
	})
}

func getPageAndSize(c *gin.Context, defaultPage, defaultPageSize int) (int, int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(defaultPage)))
	if err != nil || page < 1 {
		page = defaultPage
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", strconv.Itoa(defaultPageSize)))
	if err != nil || pageSize < 1 {
		pageSize = defaultPageSize
	}
	return page, pageSize
}

func GetPaginatedTracking(ctx context.Context, page int, pageSize int) ([]models.TrackingInformation, error) {
	offset := (page - 1) * pageSize

	err := GetDB().NewSelect().
		Model(&tracking).
		Limit(pageSize).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return tracking, nil
}

func GetTrackingByKey(c *gin.Context) {
	if GetDB() == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "false",
			Message: "Database connection is not initialized",
		})
		return
	}
	query := GetDB().NewSelect().Model(&tracking)

	if c.Query("type") != "" {
		query = query.Where(`LOWER("type") = ?`, strings.ToLower(c.Query("type")))
	}

	if c.Query("symbol") != "" {
		query = query.WhereOr(`LOWER("symbol") = ?`, strings.ToLower(c.Query("symbol")))
	}

	err := query.Scan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "false",
			Message: "Error getting tracking information",
		})
		return
	}

	if len(tracking) == 0 {
		c.JSON(http.StatusNotFound, Response{
			Status:  "false",
			Message: "No tracking found with the provided type token value",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "true",
		Message: "Found tracking successfully!",
		Data:    tracking,
	})
}

func DeleteTrackingTransaction(c *gin.Context) {
	// Check for database connection
	if GetDB() == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "false",
			Message: "Database connection is not initialized",
		})
		return
	}

	transactionToDelete := strings.ToLower(c.Query("transaction"))
	if transactionToDelete == "" {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "false",
			Message: "Transaction hash is required",
		})
		return
	}

	query := GetDB().
		NewDelete().
		Model(&models.TrackingInformation{}).
		Where(`LOWER("transactionHash") = ?`, transactionToDelete)

	// Execute the query
	res, err := query.Exec(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "false",
			Message: "Error deleting transaction hash",
		})
		log.Printf("Error during delete: %v", err)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Status:  "false",
			Message: "No hash found with the provided transaction value",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "true",
		Message: "Deleted transaction hash successfully!",
	})
}
