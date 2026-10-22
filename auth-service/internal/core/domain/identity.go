package domain

// Identity is the verified Google identity extracted from an ID token.
type Identity struct {
	Email         string
	EmailVerified bool
	Name          string
	PictureURL    string
}
