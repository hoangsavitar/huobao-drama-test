package models

import "time"

// FramePrompt 帧提示词存储表
type FramePrompt struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	StoryboardID uint      `gorm:"not null;index:idx_frame_prompts_storyboard" json:"storyboard_id"`
	FrameType    string    `gorm:"size:20;not null;index:idx_frame_prompts_type" json:"frame_type"` // first, key, last, panel, action
	Prompt    string  `gorm:"type:text;not null" json:"prompt"`
	Layout    *string `gorm:"size:50" json:"layout,omitempty"` // panel/action only, e.g. horizontal_3
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FramePrompt) TableName() string {
	return "frame_prompts"
}

// FrameType 帧类型常量
const (
	FrameTypeFirst  = "first"
	FrameTypeKey    = "key"
	FrameTypeLast   = "last"
	FrameTypePanel  = "panel"
	FrameTypeAction = "action"
)
