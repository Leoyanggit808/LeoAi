package service

//代码核心要点：

//GenerateFromPassword把用户输入的明文密码（password）安全地加密成一个无法反向破解的哈希字符串
//bcrypt 是目前业界最推荐的密码存储方式之一（比 MD5、SHA1 安全得多），它有三个核心特点：
//加盐（Salt）：每次生成的哈希都不一样，防止彩虹表攻击。
//慢：故意设计得很慢，让黑客暴力破解成本极高。
//自带版本和成本信息：生成的哈希字符串里已经包含了算法版本和 cost 参数，后面验证时不需要额外存 salt。

import (
	"LeoAi/internal/model"
	"LeoAi/internal/repository"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(username, email, password string) error {

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err // 或包装成自定义错误
	}
	/*参数含义你代码里的值说明password明文密码[]byte(password)先把 Go 的 string 转成 []byte，
	因为 bcrypt 只接受字节切片cost成本因子（Work Factor）14最关键的参数！决定哈希计算要花多少时间
	cost = 14 是什么意思？
	cost 的取值范围是 4 ~ 31（默认是 10）。
	每增加 1，计算时间大约翻一倍。
	14 是目前很多生产项目推荐的值（2025-2026 年主流）：
	够慢，能有效抵抗 GPU 暴力破解
	但又不会让用户登录时感觉到明显卡顿*/
	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
	}
	return s.repo.Create(user)
}

func (s *UserService) Login(email, password string) (*model.User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("密码错误")
	}
	return user, nil
}
