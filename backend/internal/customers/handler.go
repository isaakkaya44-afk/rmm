package customers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rmm-platform/backend/internal/auth"
	"github.com/rmm-platform/backend/internal/shared/config"
)

type Customer struct {
	ID           string  `json:"id" db:"id"`
	Name         string  `json:"name" db:"name"`
	ContactName  *string `json:"contact_name,omitempty" db:"contact_name"`
	ContactEmail *string `json:"contact_email,omitempty" db:"contact_email"`
	ContactPhone *string `json:"contact_phone,omitempty" db:"contact_phone"`
	Address      *string `json:"address,omitempty" db:"address"`
	IsActive     bool    `json:"is_active" db:"is_active"`
	CreatedAt    string  `json:"created_at" db:"created_at"`
	UpdatedAt    string  `json:"updated_at" db:"updated_at"`
}

type CreateCustomerRequest struct {
	Name         string  `json:"name" binding:"required"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Address      *string `json:"address,omitempty"`
}

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List() ([]Customer, error) {
	var customers []Customer
	err := r.db.Select(&customers, `SELECT * FROM customers WHERE is_active = TRUE ORDER BY name`)
	return customers, err
}

func (r *Repository) GetByID(id string) (*Customer, error) {
	var c Customer
	err := r.db.Get(&c, `SELECT * FROM customers WHERE id = $1`, id)
	return &c, err
}

func (r *Repository) Create(req *CreateCustomerRequest) (*Customer, error) {
	var id string
	err := r.db.Get(&id, `INSERT INTO customers (name, contact_name, contact_email, contact_phone, address) 
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		req.Name, req.ContactName, req.ContactEmail, req.ContactPhone, req.Address)
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func RegisterRoutes(rg *gin.RouterGroup, db *sqlx.DB) {
	repo := NewRepository(db)
	mid := auth.NewMiddleware(&config.Load().JWT)

	customers := rg.Group("/customers")
	customers.Use(mid.RequireAuth())
	{
		customers.GET("", func(c *gin.Context) {
			list, err := repo.List()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list customers"})
				return
			}
			c.JSON(http.StatusOK, list)
		})

		customers.GET("/:id", func(c *gin.Context) {
			cust, err := repo.GetByID(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
				return
			}
			c.JSON(http.StatusOK, cust)
		})

		customers.POST("", func(c *gin.Context) {
			var req CreateCustomerRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			cust, err := repo.Create(&req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, cust)
		})
	}
}
