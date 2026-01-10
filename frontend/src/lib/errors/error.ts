export type AuthErrorStatus = 400 | 401 | 404 | 409 | 428 | 429 | 500;

export class AuthError extends Error {
	readonly status: AuthErrorStatus;

	constructor(status: AuthErrorStatus, message: string) {
		super(message);
		this.name = 'AuthError';
		this.status = status;

		if (Error.captureStackTrace) {
			Error.captureStackTrace(this, AuthError);
		}
	}
}
