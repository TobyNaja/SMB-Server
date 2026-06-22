package domain

type User struct {
	Username string `json:"username"`
	UID      string `json:"uid"`
	Fullname string `json:"fullname"`
	Disabled bool   `json:"disabled"`
}

type Group struct {
	Name string `json:"name"`
}
