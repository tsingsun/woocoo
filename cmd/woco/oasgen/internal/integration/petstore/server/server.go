// Code generated by woco, DO NOT EDIT.

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/petstore"
)

// RegisterPetHandlers creates http.Handler with routing matching OpenAPI spec.
func RegisterPetHandlers(router *gin.Engine, si petstore.PetServer) *gin.Engine {
	router.POST("/pet", wrapAddPet(si))
	router.DELETE("/pet/{petId}", wrapDeletePet(si))
	router.GET("/pet/findByStatus", wrapFindPetsByStatus(si))
	router.GET("/pet/findByTags", wrapFindPetsByTags(si))
	router.GET("/pet/{petId}", wrapGetPetById(si))
	router.PUT("/pet", wrapUpdatePet(si))
	router.POST("/pet/{petId}", wrapUpdatePetWithForm(si))
	router.POST("/pet/{petId}/uploadImage", wrapUploadFile(si))
	return router
}

func wrapAddPet(si petstore.PetServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.AddPetRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.AddPet(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapDeletePet(si petstore.PetServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.DeletePetRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := si.DeletePet(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func wrapFindPetsByStatus(si petstore.PetServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.FindPetsByStatusRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.FindPetsByStatus(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapFindPetsByTags(si petstore.PetServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.FindPetsByTagsRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.FindPetsByTags(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapGetPetById(si petstore.PetServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.GetPetByIdRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.GetPetById(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapUpdatePet(si petstore.PetServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.UpdatePetRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.UpdatePet(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapUpdatePetWithForm(si petstore.PetServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.UpdatePetWithFormRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := si.UpdatePetWithForm(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func wrapUploadFile(si petstore.PetServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.UploadFileRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.UploadFile(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// RegisterStoreHandlers creates http.Handler with routing matching OpenAPI spec.
func RegisterStoreHandlers(router *gin.Engine, si petstore.StoreServer) *gin.Engine {
	router.DELETE("/store/order/{orderId}", wrapDeleteOrder(si))
	router.GET("/store/inventory", wrapGetInventory(si))
	router.GET("/store/order/{orderId}", wrapGetOrderById(si))
	router.POST("/store/order", wrapPlaceOrder(si))
	return router
}

func wrapDeleteOrder(si petstore.StoreServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.DeleteOrderRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := si.DeleteOrder(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func wrapGetInventory(si petstore.StoreServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		resp, err := si.GetInventory(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapGetOrderById(si petstore.StoreServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.GetOrderByIdRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.GetOrderById(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapPlaceOrder(si petstore.StoreServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.PlaceOrderRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.PlaceOrder(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// RegisterUserHandlers creates http.Handler with routing matching OpenAPI spec.
func RegisterUserHandlers(router *gin.Engine, si petstore.UserServer) *gin.Engine {
	router.POST("/user", wrapCreateUser(si))
	router.POST("/user/createWithArray", wrapCreateUsersWithArrayInput(si))
	router.POST("/user/createWithList", wrapCreateUsersWithListInput(si))
	router.DELETE("/user/{username}", wrapDeleteUser(si))
	router.GET("/user/{username}", wrapGetUserByName(si))
	router.GET("/user/login", wrapLoginUser(si))
	router.GET("/user/logout", wrapLogoutUser(si))
	router.PUT("/user/{username}", wrapUpdateUser(si))
	return router
}

func wrapCreateUser(si petstore.UserServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.CreateUserRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := si.CreateUser(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func wrapCreateUsersWithArrayInput(si petstore.UserServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		err := si.CreateUsersWithArrayInput(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func wrapCreateUsersWithListInput(si petstore.UserServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		err := si.CreateUsersWithListInput(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func wrapDeleteUser(si petstore.UserServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.DeleteUserRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := si.DeleteUser(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func wrapGetUserByName(si petstore.UserServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.GetUserByNameRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.GetUserByName(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapLoginUser(si petstore.UserServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.LoginUserRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := si.LoginUser(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func wrapLogoutUser(si petstore.UserServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		err := si.LogoutUser(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}

func wrapUpdateUser(si petstore.UserServer) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req petstore.UpdateUserRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := si.UpdateUser(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	}
}