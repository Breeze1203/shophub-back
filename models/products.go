package models

import (
	"time"

	"gorm.io/gorm"
)

// 商家信息表
type MerchantInfo struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"uniqueIndex;not null" json:"user_id"`                    // 关联用户表
	ShopName     string         `gorm:"type:varchar(200);not null;index" json:"shop_name"`      // 店铺名称
	ShopLogo     string         `gorm:"type:varchar(500)" json:"shop_logo"`                     // 店铺Logo
	Description  string         `gorm:"type:text" json:"description"`                           // 店铺简介
	ContactPhone string         `gorm:"type:varchar(50)" json:"contact_phone"`                  // 联系电话
	Address      string         `gorm:"type:varchar(500)" json:"address"`                       // 地址
	Status       string         `gorm:"type:varchar(20);default:'pending';index" json:"status"` // pending/approved/rejected/suspended
	Rating       float64        `gorm:"type:decimal(3,2);default:5.0" json:"rating"`            // 店铺评分
	SalesCount   int            `gorm:"default:0" json:"sales_count"`                           // 总销售量
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	// 关联
	User User  `gorm:"foreignKey:UserID" json:"user"`
	Pets []Pet `gorm:"foreignKey:MerchantID" json:"-"`
}

// 宠物分类表（平台统一管理）
type PetCategory struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	ParentID    *uint          `gorm:"index" json:"parent_id"`
	Sort        int            `gorm:"default:0" json:"sort"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	// 关联
	Parent   *PetCategory  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []PetCategory `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Pets     []Pet         `gorm:"foreignKey:CategoryID" json:"-"`
}

// 宠物商品表（商家上传）
type Pet struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	MerchantID     uint   `gorm:"not null;index" json:"merchant_id"` // 商家ID
	CategoryID     uint   `gorm:"not null;index" json:"category_id"`
	Name           string `gorm:"type:varchar(200);not null;index" json:"name"`
	ScientificName string `gorm:"type:varchar(200)" json:"scientific_name"`
	SKU            string `gorm:"type:varchar(100);index" json:"sku"` // 商家自定义SKU
	Description    string `gorm:"type:text" json:"description"`
	Origin         string `gorm:"type:varchar(100)" json:"origin"`
	Gender         string `gorm:"type:varchar(20)" json:"gender"` // male/female/unknown
	AgeRange       string `gorm:"type:varchar(50)" json:"age_range"`
	Size           string `gorm:"type:varchar(50)" json:"size"`
	Color          string `gorm:"type:varchar(100)" json:"color"`

	// 价格相关（单位：分）
	OriginalPrice int64 `gorm:"not null" json:"original_price"`      // 原价
	CurrentPrice  int64 `gorm:"not null;index" json:"current_price"` // 当前售价
	CostPrice     int64 `gorm:"default:0" json:"cost_price"`         // 成本价（仅商家可见）

	// 库存
	Stock      int `gorm:"default:0;index" json:"stock"`
	StockWarn  int `gorm:"default:5" json:"stock_warn"`
	SalesCount int `gorm:"default:0" json:"sales_count"`
	ViewCount  int `gorm:"default:0" json:"view_count"`

	// 状态
	Status       string `gorm:"type:varchar(20);default:'pending';index" json:"status"` // pending/approved/on_sale/sold_out/off_shelf/rejected
	RejectReason string `gorm:"type:text" json:"reject_reason,omitempty"`               // 拒绝原因
	IsRecommend  bool   `gorm:"default:false;index" json:"is_recommend"`                // 平台推荐
	IsNew        bool   `gorm:"default:false;index" json:"is_new"`
	IsHot        bool   `gorm:"default:false;index" json:"is_hot"`
	Sort         int    `gorm:"default:0" json:"sort"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Merchant       MerchantInfo       `gorm:"foreignKey:MerchantID" json:"merchant"`
	Category       PetCategory        `gorm:"foreignKey:CategoryID" json:"category"`
	Images         []PetImage         `gorm:"foreignKey:PetID" json:"images,omitempty"`
	Specifications []PetSpecification `gorm:"foreignKey:PetID" json:"specifications,omitempty"`
	Discounts      []PetDiscount      `gorm:"foreignKey:PetID" json:"discounts,omitempty"`
}

// 宠物图片表
type PetImage struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	PetID     uint           `gorm:"not null;index" json:"pet_id"`
	ImageURL  string         `gorm:"type:varchar(500);not null" json:"image_url"`
	Sort      int            `gorm:"default:0" json:"sort"`
	IsMain    bool           `gorm:"default:false" json:"is_main"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Pet Pet `gorm:"foreignKey:PetID" json:"-"`
}

// 宠物规格/属性表
type PetSpecification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PetID     uint      `gorm:"not null;index:idx_pet_spec" json:"pet_id"`
	SpecKey   string    `gorm:"type:varchar(100);not null;index:idx_pet_spec" json:"spec_key"`
	SpecValue string    `gorm:"type:varchar(500);not null" json:"spec_value"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Pet Pet `gorm:"foreignKey:PetID" json:"-"`
}

// 折扣活动表（商家创建）
type Discount struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	MerchantID  uint           `gorm:"not null;index" json:"merchant_id"` // 商家ID
	Name        string         `gorm:"type:varchar(200);not null" json:"name"` // 
	Type        string         `gorm:"type:varchar(20);not null;index" json:"type"` // percentage/fixed/newprice
	Value       int64          `gorm:"not null" json:"value"`
	StartTime   time.Time      `gorm:"index" json:"start_time"` //开始时间
	EndTime     time.Time      `gorm:"index" json:"end_time"` // 结束时间
	Status      string         `gorm:"type:varchar(20);default:'active';index" json:"status"` // active/inactive/expired
	MinAmount   int64          `gorm:"default:0" json:"min_amount"` 
	MaxDiscount int64          `gorm:"default:0" json:"max_discount"`
	UsageLimit  int            `gorm:"default:0" json:"usage_limit"`
	UsedCount   int            `gorm:"default:0" json:"used_count"`
	Description string         `gorm:"type:text" json:"description"` // 活动描述
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Merchant     MerchantInfo  `gorm:"foreignKey:MerchantID" json:"merchant"`
	PetDiscounts []PetDiscount `gorm:"foreignKey:DiscountID" json:"-"`
}

// 宠物折扣关联表
type PetDiscount struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	PetID      uint      `gorm:"not null;index:idx_pet_discount" json:"pet_id"`
	DiscountID uint      `gorm:"not null;index:idx_pet_discount" json:"discount_id"`
	CreatedAt  time.Time `json:"created_at"`

	Pet      Pet      `gorm:"foreignKey:PetID" json:"-"`
	Discount Discount `gorm:"foreignKey:DiscountID" json:"-"`
}

// 优惠券表（商家或平台创建）
type Coupon struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	MerchantID   *uint          `gorm:"index" json:"merchant_id,omitempty"` // NULL表示平台券，有值表示商家券
	Code         string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name         string         `gorm:"type:varchar(200);not null" json:"name"`
	Type         string         `gorm:"type:varchar(20);not null;index" json:"type"` // percentage/fixed
	Value        int64          `gorm:"not null" json:"value"`
	MinAmount    int64          `gorm:"default:0" json:"min_amount"`
	MaxDiscount  int64          `gorm:"default:0" json:"max_discount"`
	TotalCount   int            `gorm:"not null" json:"total_count"`
	UsedCount    int            `gorm:"default:0" json:"used_count"`
	PerUserLimit int            `gorm:"default:1" json:"per_user_limit"`
	StartTime    time.Time      `gorm:"index" json:"start_time"`
	EndTime      time.Time      `gorm:"index" json:"end_time"`
	Status       string         `gorm:"type:varchar(20);default:'active';index" json:"status"`
	Description  string         `gorm:"type:text" json:"description"`
	Scope        string         `gorm:"type:varchar(20);default:'all'" json:"scope"` // all/category/product
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Merchant        *MerchantInfo    `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
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

// 用户优惠券表（客户领取的优惠券）
type UserCoupon struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index:idx_user_coupon" json:"user_id"`
	CouponID  uint           `gorm:"not null;index:idx_user_coupon" json:"coupon_id"`
	Status    string         `gorm:"type:varchar(20);default:'unused';index" json:"status"` // unused/used/expired
	UsedAt    *time.Time     `json:"used_at,omitempty"`
	OrderID   *uint          `gorm:"index" json:"order_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User   User   `gorm:"foreignKey:UserID" json:"user"`
	Coupon Coupon `gorm:"foreignKey:CouponID" json:"coupon"`
}

// 收藏表（客户收藏商品）
type Favorite struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index:idx_user_pet" json:"user_id"`
	PetID     uint           `gorm:"not null;index:idx_user_pet" json:"pet_id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user"`
	Pet  Pet  `gorm:"foreignKey:PetID" json:"pet"`
}

// 购物车表
type Cart struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index:idx_user_pet_cart" json:"user_id"`
	PetID     uint           `gorm:"not null;index:idx_user_pet_cart" json:"pet_id"`
	Quantity  int            `gorm:"default:1;not null" json:"quantity"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user"`
	Pet  Pet  `gorm:"foreignKey:PetID" json:"pet"`
}

// 商家关注表（客户关注商家）
type MerchantFollow struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"not null;index:idx_user_merchant" json:"user_id"`
	MerchantID uint           `gorm:"not null;index:idx_user_merchant" json:"merchant_id"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	User       User           `gorm:"foreignKey:UserID" json:"user"`
	Merchant   MerchantInfo   `gorm:"foreignKey:MerchantID" json:"merchant"`
}

/* ============================================
   测试数据 SQL（MySQL）- 多商家平台版本
   ============================================ */

/*
-- 1. 用户数据（1个管理员，3个商家，5个客户）
INSERT INTO users (id, email, username, password, provider, type, avatar, created_at, updated_at) VALUES
(1, 'admin@example.com', 'admin', '$2a$10$hash...', 'local', 'admin', '/avatars/admin.jpg', NOW(), NOW()),
(2, 'merchant1@example.com', 'exotic_pets_shop', '$2a$10$hash...', 'local', 'merchant', '/avatars/m1.jpg', NOW(), NOW()),
(3, 'merchant2@example.com', 'reptile_world', '$2a$10$hash...', 'local', 'merchant', '/avatars/m2.jpg', NOW(), NOW()),
(4, 'merchant3@example.com', 'spider_heaven', '$2a$10$hash...', 'local', 'merchant', '/avatars/m3.jpg', NOW(), NOW()),
(5, 'customer1@example.com', 'john_doe', '$2a$10$hash...', 'google', 'client', '/avatars/c1.jpg', NOW(), NOW()),
(6, 'customer2@example.com', 'jane_smith', '$2a$10$hash...', 'local', 'client', '/avatars/c2.jpg', NOW(), NOW()),
(7, 'customer3@example.com', 'bob_wilson', '$2a$10$hash...', 'github', 'client', '/avatars/c3.jpg', NOW(), NOW()),
(8, 'customer4@example.com', 'alice_brown', '$2a$10$hash...', 'local', 'client', '/avatars/c4.jpg', NOW(), NOW()),
(9, 'customer5@example.com', 'charlie_davis', '$2a$10$hash...', 'facebook', 'client', '/avatars/c5.jpg', NOW(), NOW());

-- 2. 商家信息
INSERT INTO merchant_infos (id, user_id, shop_name, shop_logo, description, contact_phone, address, status, rating, sales_count, created_at, updated_at) VALUES
(1, 2, '异宠天堂', '/shops/shop1.jpg', '专业异宠繁殖基地，提供各类守宫、角蛙等爬宠', '13800138001', '北京市朝阳区XX路XX号', 'approved', 4.8, 256, NOW(), NOW()),
(2, 3, '爬行世界', '/shops/shop2.jpg', '专注爬行类宠物销售，蛇类、蜥蜴、陆龟应有尽有', '13800138002', '上海市浦东新区XX路XX号', 'approved', 4.9, 189, NOW(), NOW()),
(3, 4, '蜘蛛天堂', '/shops/shop3.jpg', '全球捕鸟蛛品种集散地，品种齐全', '13800138003', '广州市天河区XX路XX号', 'approved', 4.7, 143, NOW(), NOW());

-- 3. 宠物分类（与之前相同）
INSERT INTO pet_categories (id, name, description, parent_id, sort, is_active, created_at, updated_at) VALUES
(1, '爬行类', '蜥蜴、蛇、龟等爬行动物', NULL, 1, 1, NOW(), NOW()),
(2, '两栖类', '蛙、蝾螈等两栖动物', NULL, 2, 1, NOW(), NOW()),
(3, '节肢类', '蜘蛛、蝎子、昆虫等节肢动物', NULL, 3, 1, NOW(), NOW()),
(4, '哺乳类', '刺猬、蜜袋鼯等小型哺乳动物', NULL, 4, 1, NOW(), NOW()),
(5, '守宫', '各种守宫品种', 1, 1, 1, NOW(), NOW()),
(6, '蛇类', '各种宠物蛇', 1, 2, 1, NOW(), NOW()),
(7, '陆龟', '各种陆龟品种', 1, 3, 1, NOW(), NOW()),
(8, '角蛙', '各种角蛙品种', 2, 1, 1, NOW(), NOW()),
(9, '捕鸟蛛', '各种捕鸟蛛品种', 3, 1, 1, NOW(), NOW());

-- 4. 宠物商品（不同商家上传）
INSERT INTO pets (id, merchant_id, category_id, name, scientific_name, sku, description, origin, gender, age_range, size, color, original_price, current_price, cost_price, stock, stock_warn, sales_count, view_count, status, is_recommend, is_new, is_hot, sort, created_at, updated_at) VALUES
-- 商家1的商品
(1, 1, 5, '豹纹守宫', 'Eublepharis macularius', 'M1-LP-001', '新手入门首选，性格温顺，饲养简单', '人工繁殖', 'unknown', '亚成体', '中型', '高黄豹纹', 50000, 45000, 30000, 15, 5, 89, 1523, 'on_sale', 1, 0, 1, 10, NOW(), NOW()),
(2, 1, 5, '肥尾守宫', 'Hemitheconyx caudicinctus', 'M1-LP-002', '非洲肥尾守宫，尾巴粗壮可爱', '人工繁殖', 'female', '亚成体', '中型', '原色', 68000, 68000, 45000, 8, 5, 34, 876, 'on_sale', 1, 1, 0, 9, NOW(), NOW()),
(3, 1, 8, '霸王角蛙', 'Ceratophrys ornata', 'M1-FR-001', '超大型角蛙，食欲旺盛', '人工繁殖', 'unknown', '亚成体', '大型', '绿色', 25000, 22000, 15000, 20, 5, 112, 1876, 'on_sale', 0, 1, 1, 8, NOW(), NOW()),
(4, 1, 8, '钟角蛙', 'Ceratophrys cranwelli', 'M1-FR-002', '中型角蛙，多种颜色可选', '人工繁殖', 'unknown', '幼体', '中型', '奶油黄', 18000, 16000, 10000, 25, 5, 78, 1456, 'on_sale', 0, 1, 0, 7, NOW(), NOW()),

-- 商家2的商品
(5, 2, 5, '睫角守宫', 'Correlophus ciliatus', 'M2-LP-003', '树栖守宫，可以爬玻璃', '人工繁殖', 'male', '成体', '小型', '火焰纹', 38000, 35000, 22000, 12, 5, 67, 1245, 'on_sale', 0, 0, 1, 6, NOW(), NOW()),
(6, 2, 6, '玉米蛇', 'Pantherophis guttatus', 'M2-SN-001', '最适合新手的宠物蛇', '人工繁殖', 'unknown', '幼体', '中型', '白化红', 45000, 42000, 28000, 10, 3, 56, 998, 'on_sale', 1, 0, 0, 5, NOW(), NOW()),
(7, 2, 6, '球蟒', 'Python regius', 'M2-SN-002', '体型适中的蟒蛇，花纹多样', '人工繁殖', 'female', '亚成体', '中型', '香蕉基因', 128000, 118000, 80000, 5, 3, 23, 765, 'on_sale', 1, 1, 1, 4, NOW(), NOW()),
(8, 2, 7, '赫曼陆龟', 'Testudo hermanni', 'M2-TR-001', '小型陆龟，适合室内饲养', '人工繁殖', 'unknown', '幼体', '小型', '标准色', 88000, 88000, 60000, 6, 3, 45, 1123, 'on_sale', 1, 0, 0, 3, NOW(), NOW()),

-- 商家3的商品
(9, 3, 9, '智利红玫瑰', 'Grammostola rosea', 'M3-SP-001', '最温顺的捕鸟蛛之一', '智利', 'unknown', '亚成体', '中型', '粉红色', 15000, 15000, 8000, 18, 5, 91, 1678, 'on_sale', 1, 0, 1, 2, NOW(), NOW()),
(10, 3, 9, '墨西哥红膝头', 'Brachypelma smithi', 'M3-SP-002', '经典品种，颜色鲜艳', '墨西哥', 'unknown', '成体', '大型', '黑红配色', 35000, 32000, 20000, 8, 3, 43, 892, 'on_sale', 1, 1, 0, 1, NOW(), NOW()),
(11, 3, 9, '巴西白膝头', 'Acanthoscurria geniculata', 'M3-SP-003', '大型地栖蜘蛛，生长快', '巴西', 'unknown', '幼体', '大型', '黑白条纹', 28000, 25000, 15000, 12, 3, 28, 567, 'on_sale', 0, 1, 0, 0, NOW(), NOW());

-- 5. 宠物图片
INSERT INTO pet_images (pet_id, image_url, sort, is_main, created_at) VALUES
(1, '/uploads/m1/leopard-gecko-1.jpg', 1, 1, NOW()),
(1, '/uploads/m1/leopard-gecko-2.jpg', 2, 0, NOW()),
(2, '/uploads/m1/african-fat-tail-1.jpg', 1, 1, NOW()),
(3, '/uploads/m1/horned-frog-1.jpg', 1, 1, NOW()),
(4, '/uploads/m1/cranwell-frog-1.jpg', 1, 1, NOW()),
(5, '/uploads/m2/crested-gecko-1.jpg', 1, 1, NOW()),
(6, '/uploads/m2/corn-snake-1.jpg', 1, 1, NOW()),
(7, '/uploads/m2/ball-python-1.jpg', 1, 1, NOW()),
(8, '/uploads/m2/hermann-tortoise-1.jpg', 1, 1, NOW()),
(9, '/uploads/m3/rose-tarantula-1.jpg', 1, 1, NOW()),
(10, '/uploads/m3/redknee-tarantula-1.jpg', 1, 1, NOW()),
(11, '/uploads/m3/brazilian-white-1.jpg', 1, 1, NOW());

-- 6. 规格属性
INSERT INTO pet_specifications (pet_id, spec_key, spec_value, sort, created_at, updated_at) VALUES
(1, '饲养难度', '★☆☆☆☆ 新手友好', 1, NOW(), NOW()),
(1, '成体体长', '18-25cm', 2, NOW(), NOW()),
(1, '预期寿命', '10-20年', 3, NOW(), NOW()),
(1, '温度要求', '26-32℃', 4, NOW(), NOW()),
(1, '食物', '蟋蟀、面包虫、杜比亚', 5, NOW(), NOW()),
(2, '饲养难度', '★☆☆☆☆ 新手友好', 1, NOW(), NOW()),
(2, '成体体长', '18-23cm', 2, NOW(), NOW()),
(2, '预期寿命', '15-20年', 3, NOW(), NOW()),
(3, '饲养难度', '★☆☆☆☆ 超级简单', 1, NOW(), NOW()),
(3, '成体体长', '10-12cm', 2, NOW(), NOW()),
(3, '预期寿命', '5-8年', 3, NOW(), NOW()),
(5, '饲养难度', '★★☆☆☆ 容易', 1, NOW(), NOW()),
(5, '成体体长', '15-20cm', 2, NOW(), NOW()),
(6, '饲养难度', '★☆☆☆☆ 新手首选', 1, NOW(), NOW()),
(6, '成体体长', '120-150cm', 2, NOW(), NOW()),
(7, '饲养难度', '★★☆☆☆ 容易', 1, NOW(), NOW()),
(7, '成体体长', '100-150cm', 2, NOW(), NOW()),
(9, '饲养难度', '★☆☆☆☆ 新手友好', 1, NOW(), NOW()),
(9, '成体腿展', '12-15cm', 2, NOW(), NOW()),
(9, '预期寿命', '10-15年（雌性）', 3, NOW(), NOW()),
(10, '饲养难度', '★★☆☆☆ 容易', 1, NOW(), NOW()),
(10, '成体腿展', '15-18cm', 2, NOW(), NOW()),
(11, '饲养难度', '★★☆☆☆ 容易', 1, NOW(), NOW()),
(11, '成体腿展', '18-22cm', 2, NOW(), NOW());

-- 7. 折扣活动（各商家创建）
INSERT INTO discounts (id, merchant_id, name, type, value, start_time, end_time, status, min_amount, max_discount, usage_limit, used_count, description, created_at, updated_at) VALUES
-- 商家1的活动
(1, 1, '店铺新品9折', 'percentage', 90, '2024-12-01 00:00:00', '2024-12-31 23:59:59', 'active', 0, 0, 0, 0, '本店所有新品享受9折优惠', NOW(), NOW()),
(2, 1, '守宫专场特惠', 'fixed', 5000, '2024-12-15 00:00:00', '2024-12-25 23:59:59', 'active', 30000, 5000, 100, 23, '购买守宫满300减50', NOW(), NOW()),
(3, 1, '豹纹守宫限时特价', 'newprice', 39900, '2024-12-10 00:00:00', '2024-12-20 23:59:59', 'active', 0, 0, 50, 15, '豹纹守宫特价399元，限量50只', NOW(), NOW()),

-- 商家2的活动
(4, 2, '爬虫类85折', 'percentage', 85, '2024-12-01 00:00:00', '2025-01-31 23:59:59', 'active', 0, 20000, 0, 0, '全店爬虫类商品85折，最高优惠200元', NOW(), NOW()),
(5, 2, '球蟒特惠季', 'fixed', 10000, '2024-12-15 00:00:00', '2025-01-15 23:59:59', 'active', 50000, 10000, 200, 8, '购买球蟒满500减100', NOW(), NOW()),

-- 商家3的活动
(6, 3, '蜘蛛全场9折', 'percentage', 90, '2024-12-01 00:00:00', '2024-12-31 23:59:59', 'active', 0, 5000, 0, 0, '所有捕鸟蛛9折，最高优惠50元', NOW(), NOW()),
(7, 3, '新品上架特价', 'newprice', 22000, '2024-12-10 00:00:00', '2024-12-20 23:59:59', 'active', 0, 0, 30, 7, '巴西白膝头新品特价220元', NOW(), NOW());

-- 8. 商品折扣关联
INSERT INTO pet_discounts (pet_id, discount_id, created_at) VALUES
-- 商家1的折扣关联
(2, 1, NOW()),  -- 肥尾守宫-新品9折
(3, 1, NOW()),  -- 霸王角蛙-新品9折
(4, 1, NOW()),  -- 钟角蛙-新品9折
(1, 2, NOW()),  -- 豹纹守宫-守宫满减
(2, 2, NOW()),  -- 肥尾守宫-守宫满减
(1, 3, NOW()),  -- 豹纹守宫-特价

-- 商家2的折扣关联
(5, 4, NOW()),  -- 睫角守宫-85折
(6, 4, NOW()),  -- 玉米蛇-85折
(7, 4, NOW()),  -- 球蟒-85折
(8, 4, NOW()),  -- 赫曼陆龟-85折
(7, 5, NOW()),  -- 球蟒-特惠

-- 商家3的折扣关联
(9, 6, NOW()),   -- 智利红玫瑰-9折
(10, 6, NOW()),  -- 墨西哥红膝头-9折
(11, 6, NOW()),  -- 巴西白膝头-9折
(11, 7, NOW());  -- 巴西白膝头-特价

-- 9. 优惠券（平台券+商家券）
INSERT INTO coupons (id, merchant_id, code, name, type, value, min_amount, max_discount, total_count, used_count, per_user_limit, start_time, end_time, status, description, scope, created_at, updated_at) VALUES
-- 平台券（merchant_id为NULL）
(1, NULL, 'PLATFORM2024', '平台新用户券', 'fixed', 3000, 10000, 3000, 10000, 1234, 1, '2024-12-01 00:00:00', '2025-03-31 23:59:59', 'active', '新用户首单满100减30', 'all', NOW(), NOW()),
(2, NULL, 'XMAS2024', '圣诞狂欢券', 'percentage', 85, 30000, 15000, 5000, 678, 1, '2024-12-20 00:00:00', '2024-12-26 23:59:59', 'active', '圣诞特惠85折，最高优惠150元', 'all', NOW(), NOW()),
(3, NULL, 'REPTILE15', '爬虫类优惠券', 'percentage', 90, 15000, 10000, 3000, 456, 2, '2024-12-01 00:00:00', '2025-01-31 23:59:59', 'active', '爬虫类商品9折，最高优惠100元', 'category', NOW(), NOW()),

-- 商家1的券
(4, 1, 'SHOP1-VIP', '异宠天堂VIP券', 'fixed', 2000, 15000, 2000, 500, 89, 2, '2024-12-01 00:00:00', '2025-02-28 23:59:59', 'active', '本店满150减20', 'all', NOW(), NOW()),
(5, 1, 'SHOP1-NEW', '新客专享', 'fixed', 1000, 5000, 1000, 1000, 234, 1, '2024-12-01 00:00:00', '2025-01-31 23:59:59', 'active', '新客户满50减10', 'all', NOW(), NOW()),

-- 商家2的券
(6, 2, 'SHOP2-SNAKE', '爬行世界蛇类券', 'fixed', 3000, 20000, 3000, 300, 67, 1, '2024-12-10 00:00:00', '2025-01-31 23:59:59', 'active', '购买蛇类满200减30', 'category', NOW(), NOW()),
(7, 2, 'SHOP2-MEMBER', '会员专享券', 'percentage', 92, 10000, 5000, 800, 123, 3, '2024-12-01 00:00:00', '2025-02-28 23:59:59', 'active', '会员专享92折', 'all', NOW(), NOW()),

-- 商家3的券
(8, 3, 'SHOP3-SPIDER', '蜘蛛天堂优惠券', 'fixed', 500, 3000, 500, 2000, 456, 2, '2024-12-01 00:00:00', '2025-01-31 23:59:59', 'active', '购买蜘蛛满30减5', 'all', NOW(), NOW());

-- 10. 优惠券分类关联
INSERT INTO category_coupons (coupon_id, category_id, created_at) VALUES
(3, 1, NOW()),  -- 平台爬虫券-爬行类
(6, 6, NOW());  -- 商家2蛇类券-蛇类

-- 11. 用户优惠券（客户领取记录）
INSERT INTO user_coupons (user_id, coupon_id, status, used_at, order_id, created_at, updated_at) VALUES
-- 客户1
(5, 1, 'used', '2024-12-15 14:23:00', 1001, '2024-12-10 10:00:00', '2024-12-15 14:23:00'),
(5, 4, 'unused', NULL, NULL, '2024-12-12 15:30:00', '2024-12-12 15:30:00'),
(5, 8, 'unused', NULL, NULL, '2024-12-13 09:20:00', '2024-12-13 09:20:00'),

-- 客户2
(6, 1, 'unused', NULL, NULL, '2024-12-11 09:15:00', '2024-12-11 09:15:00'),
(6, 3, 'used', '2024-12-16 10:30:00', 1002, '2024-12-13 16:45:00', '2024-12-16 10:30:00'),
(6, 5, 'unused', NULL, NULL, '2024-12-14 11:00:00', '2024-12-14 11:00:00'),
(6, 7, 'unused', NULL, NULL, '2024-12-15 14:20:00', '2024-12-15 14:20:00'),

-- 客户3
(7, 2, 'unused', NULL, NULL, '2024-12-16 08:00:00', '2024-12-16 08:00:00'),
(7, 6, 'unused', NULL, NULL, '2024-12-12 10:30:00', '2024-12-12 10:30:00'),

-- 客户4
(8, 3, 'unused', NULL, NULL, '2024-12-11 14:22:00', '2024-12-11 14:22:00'),
(8, 4, 'used', '2024-12-14 16:45:00', 1003, '2024-12-10 12:00:00', '2024-12-14 16:45:00'),
(8, 8, 'unused', NULL, NULL, '2024-12-13 15:10:00', '2024-12-13 15:10:00'),

-- 客户5
(9, 1, 'unused', NULL, NULL, '2024-12-15 11:30:00', '2024-12-15 11:30:00'),
(9, 7, 'unused', NULL, NULL, '2024-12-14 09:45:00', '2024-12-14 09:45:00');

-- 12. 收藏记录
INSERT INTO favorites (user_id, pet_id, created_at) VALUES
(5, 1, '2024-12-10 10:30:00'),
(5, 7, '2024-12-11 14:20:00'),
(5, 9, '2024-12-12 16:45:00'),
(6, 2, '2024-12-09 11:15:00'),
(6, 6, '2024-12-10 15:30:00'),
(6, 10, '2024-12-13 09:20:00'),
(7, 3, '2024-12-11 13:40:00'),
(7, 5, '2024-12-12 10:10:00'),
(8, 4, '2024-12-10 16:25:00'),
(8, 8, '2024-12-14 11:50:00'),
(9, 1, '2024-12-13 14:35:00'),
(9, 11, '2024-12-15 10:20:00');

-- 13. 购物车
INSERT INTO carts (user_id, pet_id, quantity, created_at, updated_at) VALUES
(5, 1, 1, '2024-12-16 10:00:00', '2024-12-16 10:00:00'),
(5, 9, 2, '2024-12-16 10:05:00', '2024-12-16 10:05:00'),
(6, 6, 1, '2024-12-16 11:20:00', '2024-12-16 11:20:00'),
(7, 3, 1, '2024-12-16 09:30:00', '2024-12-16 09:30:00'),
(7, 4, 2, '2024-12-16 09:35:00', '2024-12-16 09:35:00'),
(8, 7, 1, '2024-12-16 14:15:00', '2024-12-16 14:15:00'),
(9, 10, 1, '2024-12-16 15:40:00', '2024-12-16 15:40:00');

-- 14. 商家关注
INSERT INTO merchant_follows (user_id, merchant_id, created_at) VALUES
(5, 1, '2024-12-08 10:00:00'),
(5, 3, '2024-12-09 14:20:00'),
(6, 1, '2024-12-07 11:30:00'),
(6, 2, '2024-12-10 15:45:00'),
(7, 2, '2024-12-09 09:20:00'),
(7, 3, '2024-12-11 16:30:00'),
(8, 1, '2024-12-10 13:15:00'),
(8, 2, '2024-12-12 10:40:00'),
(9, 2, '2024-12-11 11:50:00'),
(9, 3, '2024-12-13 14:25:00');

-- 查询示例
-- 1. 查询某商家的所有商品
SELECT p.*, m.shop_name
FROM pets p
JOIN merchant_infos m ON p.merchant_id = m.id
WHERE m.id = 1 AND p.status = 'on_sale';

-- 2. 查询某用户的购物车（含商品和商家信息）
SELECT c.*, p.name, p.current_price, m.shop_name
FROM carts c
JOIN pets p ON c.pet_id = p.id
JOIN merchant_infos m ON p.merchant_id = m.id
WHERE c.user_id = 5 AND c.deleted_at IS NULL;

-- 3. 查询某商品的有效折扣
SELECT d.*
FROM discounts d
JOIN pet_discounts pd ON d.id = pd.discount_id
WHERE pd.pet_id = 1
  AND d.status = 'active'
  AND NOW() BETWEEN d.start_time AND d.end_time;

-- 4. 查询用户可用的优惠券
SELECT c.*, uc.status
FROM user_coupons uc
JOIN coupons c ON uc.coupon_id = c.id
WHERE uc.user_id = 5
  AND uc.status = 'unused'
  AND NOW() BETWEEN c.start_time AND c.end_time;

-- 5. 查询某分类下的热门商品
SELECT p.*, m.shop_name, m.rating
FROM pets p
JOIN merchant_infos m ON p.merchant_id = m.id
WHERE p.category_id = 5
  AND p.status = 'on_sale'
ORDER BY p.sales_count DESC, p.view_count DESC
LIMIT 10;

*/
