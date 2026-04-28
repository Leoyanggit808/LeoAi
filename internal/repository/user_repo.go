package repository

//UserRepository，专门负责对 User 表的增删改查操作。
//它把数据库操作封装起来，让上层（Service 层）不需要关心 SQL、GORM 具体用法，
//只需要调用 Create、FindByEmail 这些方法即可。
import (
	"LeoAi/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
	//定义了一个结构体 UserRepository。
	//它只持有一个字段：*gorm.DB（数据库连接对象）。
	//这是一种依赖注入的写法，数据库连接从外面传进来，而不是在结构体内部创建。
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	//构造函数（工厂函数）。
	//外部创建 Repository 时必须传入 *gorm.DB，例如：
	return &UserRepository{
		db: db,
	}
}

// Create 新增用户
func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// FindByEmail 按邮箱查询
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
