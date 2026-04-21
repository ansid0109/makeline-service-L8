package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	var orderService *OrderService

	log.Printf("Using MongoDB API")

	// Initialize the database
	orderService, err := initDatabase()
	if err != nil {
		log.Printf("Failed to initialize database: %s", err)
		os.Exit(1)
	}

	router := gin.Default()
	router.Use(cors.Default())
	router.Use(OrderMiddleware(orderService))
	router.POST("/order", createOrder)
	router.GET("/order/:id", getOrder)
	router.PUT("/order", updateOrder)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": os.Getenv("APP_VERSION"),
		})
	})
	router.Run(":3001")
}

// OrderMiddleware is a middleware function that injects the order service into the request context
func OrderMiddleware(orderService *OrderService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("orderService", orderService)
		c.Next()
	}
}

// Receives a new order from an external service and stores it in database
func createOrder(c *gin.Context) {
	client, ok := c.MustGet("orderService").(*OrderService)
	if !ok {
		log.Printf("Failed to get order service")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var order Order
	if err := c.BindJSON(&order); err != nil {
		log.Printf("Failed to unmarshal order: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if order.OrderID == "" {
		order.OrderID = strconv.FormatInt(time.Now().UnixMilli(), 10)
	} else {
		id, err := strconv.Atoi(order.OrderID)
		if err != nil {
			log.Printf("Failed to convert order id to int: %s", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		order.OrderID = strconv.FormatInt(int64(id), 10)
	}

	// New inbound orders start in pending state.
	order.Status = Pending

	err := client.repo.InsertOrders([]Order{order})
	if err != nil {
		log.Printf("Failed to save orders to database: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusCreated, order)
}

// Gets a single order from database by order ID
func getOrder(c *gin.Context) {
	client, ok := c.MustGet("orderService").(*OrderService)
	if !ok {
		log.Printf("Failed to get order service")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Printf("Failed to convert order id to int: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	sanitizedOrderId := strconv.FormatInt(int64(id), 10)

	order, err := client.repo.GetOrder(sanitizedOrderId)
	if err != nil {
		log.Printf("Failed to get order from database: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusOK, order)
}

// Updates the status of an order
func updateOrder(c *gin.Context) {
	client, ok := c.MustGet("orderService").(*OrderService)
	if !ok {
		log.Printf("Failed to get order service")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// unmarsal the order from the request body
	var order Order
	if err := c.BindJSON(&order); err != nil {
		log.Printf("Failed to unmarshal order: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(order.OrderID)
	if err != nil {
		log.Printf("Failed to convert order id to int: %s", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	sanitizedOrderId := strconv.FormatInt(int64(id), 10)

	sanitizedOrder := Order{
		OrderID:    sanitizedOrderId,
		CustomerID: order.CustomerID,
		Items:      order.Items,
		Status:     order.Status,
	}

	err = client.repo.UpdateOrder(sanitizedOrder)
	if err != nil {
		log.Printf("Failed to update order status: %s", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.SetAccepted("202")
}

// Gets an environment variable or exits if it is not set
func getEnvVar(varName string, fallbackVarNames ...string) string {
	value := os.Getenv(varName)
	if value == "" {
		for _, fallbackVarName := range fallbackVarNames {
			value = os.Getenv(fallbackVarName)
			if value == "" {
				break
			}
		}
		if value == "" {
			log.Printf("%s is not set", varName)
			if len(fallbackVarNames) > 0 {
				log.Printf("Tried fallback variables: %v", fallbackVarNames)
			}
			os.Exit(1)
		}
	}
	return value
}

// Initializes the database
func initDatabase() (*OrderService, error) {
	dbURI := getEnvVar("ORDER_DB_URI")
	dbName := getEnvVar("ORDER_DB_NAME")
	collectionName := getEnvVar("ORDER_DB_COLLECTION_NAME")
	dbUsername := os.Getenv("ORDER_DB_USERNAME")
	dbPassword := os.Getenv("ORDER_DB_PASSWORD")
	mongoRepo, err := NewMongoDBOrderRepo(dbURI, dbName, collectionName, dbUsername, dbPassword)
	if err != nil {
		return nil, err
	}
	return NewOrderService(mongoRepo), nil
}
