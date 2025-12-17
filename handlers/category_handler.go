package handlers

import (
	"LiteAdmin/models"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type CategoryServiceHandler struct {
	db *gorm.DB
}

func NewCategoryHandler(db *gorm.DB) *CategoryServiceHandler {
	return &CategoryServiceHandler{db: db}
}

// GetCategories 获取所有分类（树形结构）
func (h *CategoryServiceHandler) GetCategories(c echo.Context) error {
	var categories []models.PetCategory

	// 只查询顶级分类（parent_id 为 NULL）
	if err := h.db.Where("parent_id IS NULL").
		Preload("Children"). // 预加载子分类
		Order("sort ASC, id ASC").
		Find(&categories).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "获取分类失败",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    categories,
	})
}

// GetAllCategories 获取所有分类（扁平列表）
func (h *CategoryServiceHandler) GetAllCategories(c echo.Context) error {
	var categories []models.PetCategory

	// 查询所有分类
	if err := h.db.Where("is_active = ?", true).
		Order("sort ASC, id ASC").
		Find(&categories).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "获取分类失败",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    categories,
	})
}

// GetCategoryByID 根据ID获取分类详情
func (h *CategoryServiceHandler) GetCategoryByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "无效的分类ID",
		})
	}

	var category models.PetCategory
	if err := h.db.Preload("Children").
		Preload("Parent").
		First(&category, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "分类不存在",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "获取分类失败",
			"error":   err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    category,
	})
}

// CreateCategory 创建分类（管理员）
func (h *CategoryServiceHandler) CreateCategory(c echo.Context) error {
	var req struct {
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
		ParentID    *uint  `json:"parent_id"`
		Sort        int    `json:"sort"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
	}
	category := models.PetCategory{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		Sort:        req.Sort,
		IsActive:    true,
	}
	if err := h.db.Create(&category).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "创建分类失败",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "创建成功",
		"data":    category,
	})
}

// UpdateCategory 更新分类（管理员）
func (h *CategoryServiceHandler) UpdateCategory(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "无效的分类ID",
		})
	}

	var category models.PetCategory
	if err := h.db.First(&category, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"code":    404,
			"message": "分类不存在",
		})
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ParentID    *uint  `json:"parent_id"`
		Sort        int    `json:"sort"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误",
		})
	}

	// 更新字段
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.ParentID != nil {
		updates["parent_id"] = req.ParentID
	}
	if req.Sort != 0 {
		updates["sort"] = req.Sort
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&category).Updates(updates).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "更新失败",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "更新成功",
		"data":    category,
	})
}

// DeleteCategory 删除分类（软删除，管理员）
func (h *CategoryServiceHandler) DeleteCategory(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "无效的分类ID",
		})
	}

	// 检查是否有子分类
	var childCount int64
	h.db.Model(&models.PetCategory{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "该分类下有子分类，无法删除",
		})
	}

	// 检查是否有商品
	var petCount int64
	h.db.Model(&models.Pet{}).Where("category_id = ?", id).Count(&petCount)
	if petCount > 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "该分类下有商品，无法删除",
		})
	}

	if err := h.db.Delete(&models.PetCategory{}, id).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "删除失败",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "删除成功",
	})
}
