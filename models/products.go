package models

import (
	"time"

	"gorm.io/gorm"
)

// 宠物分类表
type PetCategory struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"` // 分类名称：爬行类、两栖类、节肢类等
	Description string         `gorm:"type:text" json:"description"`                       // 分类描述
	ParentID    *uint          `gorm:"index" json:"parent_id"`                             // 父分类ID，支持多级分类
	Sort        int            `gorm:"default:0" json:"sort"`                              // 排序
	IsActive    bool           `gorm:"default:true" json:"is_active"`                      // 是否启用
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Parent   *PetCategory  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []PetCategory `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Pets     []Pet         `gorm:"foreignKey:CategoryID" json:"-"`
}

// 宠物产品表（主表）
type Pet struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	CategoryID     uint   `gorm:"not null;index" json:"category_id"`            // 分类ID
	Name           string `gorm:"type:varchar(200);not null;index" json:"name"` // 宠物名称
	ScientificName string `gorm:"type:varchar(200)" json:"scientific_name"`     // 学名
	SKU            string `gorm:"type:varchar(100);uniqueIndex" json:"sku"`     // 商品编码
	Description    string `gorm:"type:text" json:"description"`                 // 详细描述
	Origin         string `gorm:"type:varchar(100)" json:"origin"`              // 产地
	Gender         string `gorm:"type:varchar(20)" json:"gender"`               // 性别：male/female/unknown
	AgeRange       string `gorm:"type:varchar(50)" json:"age_range"`            // 年龄段
	Size           string `gorm:"type:varchar(50)" json:"size"`                 // 体型大小
	Color          string `gorm:"type:varchar(100)" json:"color"`               // 颜色/品相

	// 价格相关（单位：分，避免浮点数精度问题）
	OriginalPrice int64 `gorm:"not null" json:"original_price"`      // 原价（分）
	CurrentPrice  int64 `gorm:"not null;index" json:"current_price"` // 当前售价（分）
	CostPrice     int64 `gorm:"default:0" json:"cost_price"`         // 成本价（分）

	// 库存
	Stock      int `gorm:"default:0;index" json:"stock"` // 库存数量
	StockWarn  int `gorm:"default:5" json:"stock_warn"`  // 库存预警值
	SalesCount int `gorm:"default:0" json:"sales_count"` // 销售数量
	ViewCount  int `gorm:"default:0" json:"view_count"`  // 浏览次数

	// 状态
	Status      string `gorm:"type:varchar(20);default:'on_sale';index" json:"status"` // on_sale/sold_out/off_shelf
	IsRecommend bool   `gorm:"default:false;index" json:"is_recommend"`                // 是否推荐
	IsNew       bool   `gorm:"default:false;index" json:"is_new"`                      // 是否新品
	IsHot       bool   `gorm:"default:false;index" json:"is_hot"`                      // 是否热销
	Sort        int    `gorm:"default:0" json:"sort"`                                  // 排序

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Category       PetCategory        `gorm:"foreignKey:CategoryID" json:"category"`
	Images         []PetImage         `gorm:"foreignKey:PetID" json:"images,omitempty"`
	Specifications []PetSpecification `gorm:"foreignKey:PetID" json:"specifications,omitempty"`
	Discounts      []PetDiscount      `gorm:"foreignKey:PetID" json:"discounts,omitempty"`
}

// 宠物图片表
type PetImage struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	PetID     uint           `gorm:"not null;index" json:"pet_id"`
	ImageURL  string         `gorm:"type:varchar(500);not null" json:"image_url"` // 图片URL
	Sort      int            `gorm:"default:0" json:"sort"`                       // 排序
	IsMain    bool           `gorm:"default:false" json:"is_main"`                // 是否主图
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Pet Pet `gorm:"foreignKey:PetID" json:"-"`
}

// 宠物规格/属性表
type PetSpecification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PetID     uint      `gorm:"not null;index:idx_pet_spec" json:"pet_id"`
	SpecKey   string    `gorm:"type:varchar(100);not null;index:idx_pet_spec" json:"spec_key"` // 规格键：饲养难度、寿命等
	SpecValue string    `gorm:"type:varchar(500);not null" json:"spec_value"`                  // 规格值
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Pet Pet `gorm:"foreignKey:PetID" json:"-"`
}

// 折扣活动表
type Discount struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(200);not null" json:"name"`                // 活动名称
	Type        string         `gorm:"type:varchar(20);not null;index" json:"type"`           // percentage/fixed/newprice
	Value       int64          `gorm:"not null" json:"value"`                                 // 折扣值：percentage=85表示85折，fixed=1000表示减10元，newprice=9900表示特价99元
	StartTime   time.Time      `gorm:"index" json:"start_time"`                               // 开始时间
	EndTime     time.Time      `gorm:"index" json:"end_time"`                                 // 结束时间
	Status      string         `gorm:"type:varchar(20);default:'active';index" json:"status"` // active/inactive/expired
	MinAmount   int64          `gorm:"default:0" json:"min_amount"`                           // 最低消费金额（分）
	MaxDiscount int64          `gorm:"default:0" json:"max_discount"`                         // 最大优惠金额（分），0表示不限制
	UsageLimit  int            `gorm:"default:0" json:"usage_limit"`                          // 使用次数限制，0表示不限制
	UsedCount   int            `gorm:"default:0" json:"used_count"`                           // 已使用次数
	Description string         `gorm:"type:text" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	PetDiscounts []PetDiscount `gorm:"foreignKey:DiscountID" json:"-"`
}

// 宠物折扣关联表（多对多）
type PetDiscount struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	PetID      uint      `gorm:"not null;index:idx_pet_discount" json:"pet_id"`
	DiscountID uint      `gorm:"not null;index:idx_pet_discount" json:"discount_id"`
	CreatedAt  time.Time `json:"created_at"`

	Pet      Pet      `gorm:"foreignKey:PetID" json:"-"`
	Discount Discount `gorm:"foreignKey:DiscountID" json:"-"`
}

// 优惠券表
type Coupon struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Code         string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"` // 优惠券码
	Name         string         `gorm:"type:varchar(200);not null" json:"name"`            // 优惠券名称
	Type         string         `gorm:"type:varchar(20);not null;index" json:"type"`       // percentage/fixed
	Value        int64          `gorm:"not null" json:"value"`                             // 优惠值
	MinAmount    int64          `gorm:"default:0" json:"min_amount"`                       // 最低消费金额（分）
	MaxDiscount  int64          `gorm:"default:0" json:"max_discount"`                     // 最大优惠金额（分）
	TotalCount   int            `gorm:"not null" json:"total_count"`                       // 发行总量
	UsedCount    int            `gorm:"default:0" json:"used_count"`                       // 已使用数量
	PerUserLimit int            `gorm:"default:1" json:"per_user_limit"`                   // 每人限领数量
	StartTime    time.Time      `gorm:"index" json:"start_time"`
	EndTime      time.Time      `gorm:"index" json:"end_time"`
	Status       string         `gorm:"type:varchar(20);default:'active';index" json:"status"` // active/inactive/expired
	Description  string         `gorm:"type:text" json:"description"`
	Scope        string         `gorm:"type:varchar(20);default:'all'" json:"scope"` // all/category/product
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	CategoryCoupons []CategoryCoupon `gorm:"foreignKey:CouponID" json:"-"`
	PetCoupons      []PetCoupon      `gorm:"foreignKey:CouponID" json:"-"`
}

// 优惠券分类关联表
type CategoryCoupon struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CouponID   uint      `gorm:"not null;index:idx_coupon_category" json:"coupon_id"`
	CategoryID uint      `gorm:"not null;index:idx_coupon_category" json:"category_id"`
	CreatedAt  time.Time `json:"created_at"`

	Coupon   Coupon      `gorm:"foreignKey:CouponID" json:"-"`
	Category PetCategory `gorm:"foreignKey:CategoryID" json:"-"`
}

// 优惠券商品关联表
type PetCoupon struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CouponID  uint      `gorm:"not null;index:idx_coupon_pet" json:"coupon_id"`
	PetID     uint      `gorm:"not null;index:idx_coupon_pet" json:"pet_id"`
	CreatedAt time.Time `json:"created_at"`

	Coupon Coupon `gorm:"foreignKey:CouponID" json:"-"`
	Pet    Pet    `gorm:"foreignKey:PetID" json:"-"`
}

// 用户优惠券表
type UserCoupon struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index:idx_user_coupon" json:"user_id"`
	CouponID  uint           `gorm:"not null;index:idx_user_coupon" json:"coupon_id"`
	Status    string         `gorm:"type:varchar(20);default:'unused';index" json:"status"` // unused/used/expired
	UsedAt    *time.Time     `json:"used_at,omitempty"`                                     // 使用时间
	OrderID   *uint          `gorm:"index" json:"order_id,omitempty"`                       // 关联订单ID
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Coupon Coupon `gorm:"foreignKey:CouponID" json:"coupon"`
}
