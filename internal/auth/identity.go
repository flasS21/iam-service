package auth

type Identity struct {
	KeycloakSub   string
	Email         string
	EmailVerified bool
}
