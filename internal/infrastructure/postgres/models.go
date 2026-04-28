package postgres

import "time"

type RoomModel struct {
	ID                  string    `gorm:"type:uuid;primaryKey"`
	RoomID              string    `gorm:"column:room_id;type:varchar(128);uniqueIndex;not null"`
	OwnerIdentifierHash string    `gorm:"column:owner_identifier_hash;type:char(64);not null"`
	IsDestroyed         bool      `gorm:"column:is_destroyed;not null;default:false"`
	ExpiresAt           time.Time `gorm:"column:expires_at;not null"`
	CreatedAt           time.Time `gorm:"column:created_at;not null"`
	UpdatedAt           time.Time `gorm:"column:updated_at;not null"`
}

func (RoomModel) TableName() string {
	return "rooms"
}

type RoomMemberModel struct {
	ID             string     `gorm:"type:uuid;primaryKey"`
	RoomID         string     `gorm:"column:room_id;type:varchar(128);not null;index:idx_room_members_room_identifier,unique"`
	IdentifierHash string     `gorm:"column:identifier_hash;type:char(64);not null;index:idx_room_members_room_identifier,unique"`
	JoinedAt       time.Time  `gorm:"column:joined_at;not null"`
	LeftAt         *time.Time `gorm:"column:left_at"`
}

func (RoomMemberModel) TableName() string {
	return "room_members"
}

type ClientAliasModel struct {
	ID             string    `gorm:"type:uuid;primaryKey"`
	RoomID         string    `gorm:"column:room_id;type:varchar(128);not null;index:idx_client_aliases_room_identifier,unique"`
	IdentifierHash string    `gorm:"column:identifier_hash;type:char(64);not null;index:idx_client_aliases_room_identifier,unique"`
	Alias          string    `gorm:"column:alias;type:varchar(128);not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (ClientAliasModel) TableName() string {
	return "client_aliases"
}

type MessageModel struct {
	ID                   string    `gorm:"type:uuid;primaryKey"`
	RoomID               string    `gorm:"column:room_id;type:varchar(128);not null;index"`
	Body                 string    `gorm:"column:body;type:text;not null"`
	SenderIdentifierHash string    `gorm:"column:sender_identifier_hash;type:char(64);not null"`
	SenderAlias          string    `gorm:"column:sender_alias;type:varchar(128);not null"`
	SentAt               time.Time `gorm:"column:sent_at;not null;index"`
	CreatedAt            time.Time `gorm:"column:created_at;not null"`
}

func (MessageModel) TableName() string {
	return "messages"
}
