package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

var ErrNotFound = errors.New("not found")

func listingImagesJSON(images []string) (string, error) {
	if len(images) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(images)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func applyListingImages(l *Listing, raw string) {
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &l.ImageURLs)
	}
	if len(l.ImageURLs) == 0 && l.ImageURL != "" {
		l.ImageURLs = []string{l.ImageURL}
	}
	if l.ImageURL == "" && len(l.ImageURLs) > 0 {
		l.ImageURL = l.ImageURLs[0]
	}
}

func (s *Store) CreateUser(ctx context.Context, name, email, passwordHash string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		insert into users (name, email, password_hash)
		values ($1, lower($2), $3)
		returning id, name, email, profile_image_url, bio, password_hash, created_at
	`, name, email, passwordHash).Scan(&u.ID, &u.Name, &u.Email, &u.ProfileImageURL, &u.Bio, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		select id, name, email, profile_image_url, bio, password_hash, created_at from users where email = lower($1)
	`, email).Scan(&u.ID, &u.Name, &u.Email, &u.ProfileImageURL, &u.Bio, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) FindUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		select id, name, email, profile_image_url, bio, password_hash, created_at from users where id = $1
	`, id).Scan(&u.ID, &u.Name, &u.Email, &u.ProfileImageURL, &u.Bio, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) UpdateProfile(ctx context.Context, id int64, name, bio, profileImageURL string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		update users
		set name = $2, bio = $3, profile_image_url = $4
		where id = $1
		returning id, name, email, profile_image_url, bio, password_hash, created_at
	`, id, name, bio, profileImageURL).Scan(&u.ID, &u.Name, &u.Email, &u.ProfileImageURL, &u.Bio, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) FindPublicUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		select id, name, profile_image_url, bio, created_at from users where id = $1
	`, id).Scan(&u.ID, &u.Name, &u.ProfileImageURL, &u.Bio, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) ListListings(ctx context.Context) ([]Listing, error) {
	rows, err := s.db.QueryContext(ctx, `
		select l.id, l.seller_id, u.name, l.title, l.description, l.price, l.image_url, l.image_urls, l.status, l.buyer_id, l.created_at, l.purchased_at
		from listings l
		join users u on u.id = l.seller_id
		order by l.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []Listing
	for rows.Next() {
		var l Listing
		var imageURLsRaw string
		if err := rows.Scan(&l.ID, &l.SellerID, &l.SellerName, &l.Title, &l.Description, &l.Price, &l.ImageURL, &imageURLsRaw, &l.Status, &l.BuyerID, &l.CreatedAt, &l.PurchasedAt); err != nil {
			return nil, err
		}
		applyListingImages(&l, imageURLsRaw)
		listings = append(listings, l)
	}
	return listings, rows.Err()
}

func (s *Store) CreateListing(ctx context.Context, sellerID int64, title, description string, price int, imageURL string, imageURLs []string) (Listing, error) {
	imageURLsRaw, err := listingImagesJSON(imageURLs)
	if err != nil {
		return Listing{}, err
	}
	var l Listing
	row := s.db.QueryRowContext(ctx, `
		with inserted as (
			insert into listings (seller_id, title, description, price, image_url, image_urls)
			values ($1, $2, $3, $4, $5, $6)
			returning id, seller_id, title, description, price, image_url, image_urls, status, buyer_id, created_at, purchased_at
		)
		select i.id, i.seller_id, u.name, i.title, i.description, i.price, i.image_url, i.image_urls, i.status, i.buyer_id, i.created_at, i.purchased_at
		from inserted i
		join users u on u.id = i.seller_id
	`, sellerID, title, description, price, imageURL, imageURLsRaw)
	var imageURLsReturned string
	err = row.Scan(&l.ID, &l.SellerID, &l.SellerName, &l.Title, &l.Description, &l.Price, &l.ImageURL, &imageURLsReturned, &l.Status, &l.BuyerID, &l.CreatedAt, &l.PurchasedAt)
	applyListingImages(&l, imageURLsReturned)
	return l, err
}

func (s *Store) PurchaseListing(ctx context.Context, listingID, buyerID int64, messageBody string) (PurchaseResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PurchaseResult{}, err
	}
	defer tx.Rollback()

	var l Listing
	row := tx.QueryRowContext(ctx, `
		with updated as (
			update listings
			set status = 'sold', buyer_id = $2, purchased_at = now()
			where id = $1 and status = 'available' and seller_id <> $2
			returning id, seller_id, title, description, price, image_url, image_urls, status, buyer_id, created_at, purchased_at
		)
		select u.id, u.seller_id, users.name, u.title, u.description, u.price, u.image_url, u.image_urls, u.status, u.buyer_id, u.created_at, u.purchased_at
		from updated u
		join users on users.id = u.seller_id
	`, listingID, buyerID)
	var imageURLsRaw string
	err = row.Scan(&l.ID, &l.SellerID, &l.SellerName, &l.Title, &l.Description, &l.Price, &l.ImageURL, &imageURLsRaw, &l.Status, &l.BuyerID, &l.CreatedAt, &l.PurchasedAt)
	applyListingImages(&l, imageURLsRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return PurchaseResult{}, ErrNotFound
	}
	if err != nil {
		return PurchaseResult{}, err
	}
	messageBody = "以下の商品を購入しました。\n\n商品名: " + l.Title + "\n\n" + strings.TrimPrefix(messageBody, "以下の商品を購入しました。\n\n")

	var c Conversation
	err = tx.QueryRowContext(ctx, `
		with upserted as (
			insert into conversations (listing_id, buyer_id, seller_id)
			values ($1, $2, $3)
			on conflict (listing_id, buyer_id) do update set updated_at = now()
			returning id, listing_id, buyer_id, seller_id, updated_at
		)
		select c.id, c.listing_id, c.buyer_id, c.seller_id, l.title, seller.id, seller.name, c.updated_at
		from upserted c
		join listings l on l.id = c.listing_id
		join users seller on seller.id = c.seller_id
	`, listingID, buyerID, l.SellerID).Scan(&c.ID, &c.ListingID, &c.BuyerID, &c.SellerID, &c.Title, &c.OtherUserID, &c.OtherUserName, &c.UpdatedAt)
	if err != nil {
		return PurchaseResult{}, err
	}

	var m Message
	err = tx.QueryRowContext(ctx, `
		insert into messages (conversation_id, sender_id, body, attachment_url)
		values ($1, $2, $3, '')
		returning id, conversation_id, sender_id, body, attachment_url, created_at
	`, c.ID, buyerID, messageBody).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.AttachmentURL, &m.CreatedAt)
	if err != nil {
		return PurchaseResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `update conversations set updated_at = now() where id = $1`, c.ID); err != nil {
		return PurchaseResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		insert into purchase_notifications (listing_id, seller_id, buyer_id, conversation_id)
		values ($1, $2, $3, $4)
	`, listingID, l.SellerID, buyerID, c.ID); err != nil {
		return PurchaseResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return PurchaseResult{}, err
	}
	return PurchaseResult{Listing: l, Conversation: c, Message: m}, nil
}

func (s *Store) CancelListing(ctx context.Context, listingID, sellerID int64) (Listing, error) {
	var l Listing
	row := s.db.QueryRowContext(ctx, `
		with updated as (
			update listings
			set status = 'cancelled'
			where id = $1 and seller_id = $2 and status = 'available'
			returning id, seller_id, title, description, price, image_url, image_urls, status, buyer_id, created_at, purchased_at
		)
		select u.id, u.seller_id, users.name, u.title, u.description, u.price, u.image_url, u.image_urls, u.status, u.buyer_id, u.created_at, u.purchased_at
		from updated u
		join users on users.id = u.seller_id
	`, listingID, sellerID)
	var imageURLsRaw string
	err := row.Scan(&l.ID, &l.SellerID, &l.SellerName, &l.Title, &l.Description, &l.Price, &l.ImageURL, &imageURLsRaw, &l.Status, &l.BuyerID, &l.CreatedAt, &l.PurchasedAt)
	applyListingImages(&l, imageURLsRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return Listing{}, ErrNotFound
	}
	return l, err
}

func (s *Store) StartConversation(ctx context.Context, listingID, buyerID int64) (Conversation, error) {
	var c Conversation
	err := s.db.QueryRowContext(ctx, `
		with upserted as (
			insert into conversations (listing_id, buyer_id, seller_id)
			select id, $2, seller_id from listings where id = $1 and seller_id <> $2
			on conflict (listing_id, buyer_id) do update set updated_at = now()
			returning id, listing_id, buyer_id, seller_id, updated_at
		)
		select c.id, c.listing_id, c.buyer_id, c.seller_id, l.title, u.id, u.name, c.updated_at
		from upserted c
		join listings l on l.id = c.listing_id
		join users u on u.id = c.seller_id
	`, listingID, buyerID).Scan(&c.ID, &c.ListingID, &c.BuyerID, &c.SellerID, &c.Title, &c.OtherUserID, &c.OtherUserName, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	return c, err
}

func (s *Store) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx, `
		select
			c.id,
			c.listing_id,
			c.buyer_id,
			c.seller_id,
			l.title,
			case when c.buyer_id = $1 then seller.id else buyer.id end as other_user_id,
			case when c.buyer_id = $1 then seller.name else buyer.name end as other_user_name,
			c.updated_at
		from conversations c
		join listings l on l.id = c.listing_id
		join users buyer on buyer.id = c.buyer_id
		join users seller on seller.id = c.seller_id
		where c.buyer_id = $1 or c.seller_id = $1
		order by c.updated_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := []Conversation{}
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.ListingID, &c.BuyerID, &c.SellerID, &c.Title, &c.OtherUserID, &c.OtherUserName, &c.UpdatedAt); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}

func (s *Store) CreateMessage(ctx context.Context, conversationID, senderID int64, body, attachmentURL string) (Message, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback()

	var m Message
	err = tx.QueryRowContext(ctx, `
		insert into messages (conversation_id, sender_id, body, attachment_url)
		select id, $2, $3, $4
		from conversations
		where id = $1 and (buyer_id = $2 or seller_id = $2)
		returning id, conversation_id, sender_id, body, attachment_url, created_at
	`, conversationID, senderID, body, attachmentURL).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.AttachmentURL, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, err
	}
	if _, err := tx.ExecContext(ctx, `update conversations set updated_at = now() where id = $1`, conversationID); err != nil {
		return Message{}, err
	}
	if err := tx.Commit(); err != nil {
		return Message{}, err
	}
	return m, nil
}

func (s *Store) ListMessages(ctx context.Context, conversationID, userID int64) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		select m.id, m.conversation_id, m.sender_id, m.body, m.attachment_url, m.created_at
		from messages m
		join conversations c on c.id = m.conversation_id
		where m.conversation_id = $1 and (c.buyer_id = $2 or c.seller_id = $2)
		order by m.created_at asc
	`, conversationID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.AttachmentURL, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (s *Store) ListUnreadPurchaseNotifications(ctx context.Context, sellerID int64) ([]PurchaseNotification, error) {
	rows, err := s.db.QueryContext(ctx, `
		select pn.id, pn.listing_id, pn.conversation_id, pn.buyer_id, buyer.name, l.title, pn.read_at, pn.created_at
		from purchase_notifications pn
		join users buyer on buyer.id = pn.buyer_id
		join listings l on l.id = pn.listing_id
		where pn.seller_id = $1 and pn.read_at is null
		order by pn.created_at asc
	`, sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := []PurchaseNotification{}
	for rows.Next() {
		var n PurchaseNotification
		if err := rows.Scan(&n.ID, &n.ListingID, &n.ConversationID, &n.BuyerID, &n.BuyerName, &n.Title, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (s *Store) MarkPurchaseNotificationRead(ctx context.Context, notificationID, sellerID int64) error {
	res, err := s.db.ExecContext(ctx, `
		update purchase_notifications
		set read_at = now()
		where id = $1 and seller_id = $2 and read_at is null
	`, notificationID, sellerID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
