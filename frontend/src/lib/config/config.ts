export const cfg = {
	apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:3003/api/users',
	apiHealthCheckUrl: import.meta.env.VITE_API_HEALTH_CHECK_URL || 'http://localhost:3003/api/ping'
};
