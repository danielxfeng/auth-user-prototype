import { PUBLIC_API_BASE_URL, PUBLIC_API_HEALTH_CHECK_URL } from '$env/static/public';

export const cfg = {
	apiBaseUrl: PUBLIC_API_BASE_URL || 'http://localhost:3003/api/users',
	apiHealthCheckUrl: PUBLIC_API_HEALTH_CHECK_URL || 'http://localhost:3003/api/ping'
};
