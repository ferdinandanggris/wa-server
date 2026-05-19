package models

type WhatsAppBusinessProfile struct {
	About             string   `json:"about,omitempty"`
	Address           string   `json:"address,omitempty"`
	Description       string   `json:"description,omitempty"`
	Email             string   `json:"email,omitempty"`
	ProfilePictureURL string   `json:"profile_picture_url,omitempty"`
	Websites          []string `json:"websites,omitempty"`
	Vertical          string   `json:"vertical,omitempty"`
	MessagingProduct  string   `json:"messaging_product,omitempty"`
}

type WhatsAppBusinessProfileResponse struct {
	Data []WhatsAppBusinessProfile `json:"data"`
}
