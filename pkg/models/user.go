package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	RepoHostAccounts []RepoHostAccount `json:"repo_host_accounts,omitempty" gorm:"foreignKey:UserID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u *User) Create() error {
	return database.Conn().Create(u).Error
}

func (u *User) Update() error {
	return database.Conn().Save(u).Error
}

func FindUserByID(id string) (*User, error) {
	var user User
	userUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Where("id = ?", userUUID).First(&user).Error
	return &user, err
}

func FindUserByEmail(email string) (*User, error) {
	var user User
	err := database.Conn().Where("email = ?", email).First(&user).Error
	return &user, err
}

func (u *User) GetRepoHostAccounts() ([]RepoHostAccount, error) {
	return FindRepoHostAccountsByUserID(u.ID)
}

func (u *User) GetRepoHostAccount(provider string) (*RepoHostAccount, error) {
	return FindRepoHostAccountByUserAndProvider(u.ID, provider)
}

func (u *User) HasRepoHostAccount(provider string) bool {
	_, err := u.GetRepoHostAccount(provider)
	return err == nil
}
