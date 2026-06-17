package db

import "time"

type User struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	ProfileImageURL string    `json:"profile_image_url"`
	Bio             string    `json:"bio"`
	PasswordHash    string    `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
}

type Listing struct {
	ID          int64      `json:"id"`
	SellerID    int64      `json:"seller_id"`
	SellerName  string     `json:"seller_name"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Price       int        `json:"price"`
	ImageURL    string     `json:"image_url"`
	ImageURLs   []string   `json:"image_urls"`
	Status      string     `json:"status"`
	BuyerID     *int64     `json:"buyer_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	PurchasedAt *time.Time `json:"purchased_at,omitempty"`
}

type Conversation struct {
	ID            int64     `json:"id"`
	ListingID     int64     `json:"listing_id"`
	BuyerID       int64     `json:"buyer_id"`
	SellerID      int64     `json:"seller_id"`
	Title         string    `json:"title"`
	OtherUserID   int64     `json:"other_user_id"`
	OtherUserName string    `json:"other_user_name"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	SenderID       int64     `json:"sender_id"`
	Body           string    `json:"body"`
	AttachmentURL  string    `json:"attachment_url"`
	CreatedAt      time.Time `json:"created_at"`
}

type PurchaseResult struct {
	Listing      Listing      `json:"listing"`
	Conversation Conversation `json:"conversation"`
	Message      Message      `json:"message"`
}

type PurchaseNotification struct {
	ID             int64      `json:"id"`
	ListingID      int64      `json:"listing_id"`
	ConversationID int64      `json:"conversation_id"`
	BuyerID        int64      `json:"buyer_id"`
	BuyerName      string     `json:"buyer_name"`
	Title          string     `json:"title"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
