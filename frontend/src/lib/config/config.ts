export const cfg = {
	apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:3003/api/users',
	totpUrl: import.meta.env.VITE_TOTP_URL || 'otpauth://totp/transcendencce:'
};
