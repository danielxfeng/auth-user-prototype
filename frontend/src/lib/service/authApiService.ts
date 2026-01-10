import type {
	AddNewFriendRequest,
	CreateUserRequest,
	LoginUserByIdentifierRequest,
	TwoFaChallengeRequest,
	TwoFaConfirmRequest,
	TwoFaDisableRequest,
	TwoFaSetupResponse,
	UpdateUserPasswordRequest,
	UpdateUserRequest,
	UsersResponse,
	UserWithoutTokenResponse,
	UserWithTokenResponse
} from '$lib/schemas/types.js';
import {
	AddNewFriendRequestSchema,
	CreateUserSchema,
	LoginUserByIdentifierRequestSchema,
	TwoFaChallengeRequestSchema,
	TwoFaConfirmRequestSchema,
	TwoFaDisableRequestSchema,
	TwoFaSetupResponseSchema,
	UpdateUserPasswordRequestSchema,
	UpdateUserRequestSchema,
	UsersResponseSchema,
	UserWithoutTokenResponseSchema,
	UserWithTokenResponseSchema
} from '$lib/schemas/userSchema.js';
import { STORAGE_TOKEN } from '$lib/stores/userStore.js';
import * as z from 'zod';
import { AuthError, type AuthErrorStatus } from '../errors/error.js';
import { cfg } from '$lib/config/config.js';

type MethodType = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';

const apiFetcher = async <TRequest, TResponse>(
	path: string,
	method: MethodType,
	data?: unknown,
	requestSchema?: z.ZodType<TRequest>,
	responseSchema?: z.ZodType<TResponse>,
	surpressAuthRedirect = false
): Promise<TResponse> => {
	if (data !== undefined && !requestSchema) {
		throw new AuthError(400, 'Request schema is required when data is provided');
	}

	const validateRequest = data !== undefined ? requestSchema!.safeParse(data) : undefined;

	if (validateRequest && !validateRequest.success) {
		throw new AuthError(400, validateRequest.error.message);
	}

	let token: string | null = null;
	try {
		token = localStorage.getItem(STORAGE_TOKEN) || null;
	} catch {
		// Ignore localStorage errors
	}

	const response = await fetch(`${cfg.apiBaseUrl}${path}`, {
		method,
		headers: {
			'Content-Type': 'application/json',
			...(token ? { Authorization: `Bearer ${token}` } : {})
		},
		body: data ? JSON.stringify(validateRequest!.data) : undefined
	});

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let responseData: any;

	try {
		responseData = await response.json();
	} catch {
		throw new AuthError(500, 'Invalid JSON response from server');
	}

	if (!response.ok) {
		const message =
			typeof responseData?.error === 'string' ? responseData.error : 'Unknown error occurred';

		if (!surpressAuthRedirect && response.status == 401) {
			window.location.href = '/user/reset';
		}

		throw new AuthError(response.status as AuthErrorStatus, message);
	}

	if (responseSchema) {
		const validateResponse = responseSchema.safeParse(responseData);
		if (!validateResponse.success) {
			throw new AuthError(500, `Invalid response format: ${validateResponse.error.message}`);
		}
		return validateResponse.data as TResponse;
	}

	return responseData as TResponse;
};

export const registerUser = async (request: CreateUserRequest): Promise<void> => {
	await apiFetcher<CreateUserRequest, UserWithoutTokenResponse>(
		'/',
		'POST',
		request,
		CreateUserSchema,
		UserWithoutTokenResponseSchema
	);
};

export const loginUser = async (
	request: LoginUserByIdentifierRequest
): Promise<UserWithTokenResponse | '2FA_REQUIRED'> => {
	try {
		const response = await apiFetcher<LoginUserByIdentifierRequest, UserWithTokenResponse>(
			'/loginByIdentifier',
			'POST',
			request,
			LoginUserByIdentifierRequestSchema,
			UserWithTokenResponseSchema,
			true
		);
		return response;
	} catch (error: unknown) {
		if (error instanceof AuthError && error.status === 428) {
			return '2FA_REQUIRED';
		}
		throw error;
	}
};

export const logoutUser = async (): Promise<void> => {
	await apiFetcher<undefined, undefined>('/logout', 'DELETE');
};

export const getUserProfile = async (): Promise<UserWithoutTokenResponse> => {
	return await apiFetcher<undefined, UserWithoutTokenResponse>(
		'/me',
		'GET',
		undefined,
		undefined,
		UserWithoutTokenResponseSchema
	);
};

export const updatePassword = async (
	request: UpdateUserPasswordRequest
): Promise<UserWithTokenResponse> => {
	return await apiFetcher<UpdateUserPasswordRequest, UserWithTokenResponse>(
		'/password',
		'PUT',
		request,
		UpdateUserPasswordRequestSchema,
		UserWithTokenResponseSchema
	);
};

export const updateProfile = async (
	request: UpdateUserRequest
): Promise<UserWithoutTokenResponse> => {
	return await apiFetcher<UpdateUserRequest, UserWithoutTokenResponse>(
		'/me',
		'PUT',
		request,
		UpdateUserRequestSchema,
		UserWithoutTokenResponseSchema
	);
};

export const deleteAccount = async (): Promise<void> => {
	await apiFetcher<undefined, undefined>('/me', 'DELETE');
};

export const startTwoFaSetup = async (): Promise<TwoFaSetupResponse> => {
	return await apiFetcher<undefined, TwoFaSetupResponse>(
		'/2fa/setup',
		'POST',
		undefined,
		undefined,
		TwoFaSetupResponseSchema
	);
};

export const twoFaConfirm = async (
	request: TwoFaConfirmRequest
): Promise<UserWithTokenResponse> => {
	return await apiFetcher<TwoFaConfirmRequest, UserWithTokenResponse>(
		'/2fa/confirm',
		'POST',
		request,
		TwoFaConfirmRequestSchema,
		UserWithTokenResponseSchema
	);
};

export const disableTwoFa = async (
	request: TwoFaDisableRequest
): Promise<UserWithTokenResponse> => {
	return await apiFetcher<TwoFaDisableRequest, UserWithTokenResponse>(
		'/2fa/disable',
		'PUT',
		request,
		TwoFaDisableRequestSchema,
		UserWithTokenResponseSchema
	);
};

export const twoFaChallenge = async (
	request: TwoFaChallengeRequest
): Promise<UserWithTokenResponse> => {
	return await apiFetcher<TwoFaChallengeRequest, UserWithTokenResponse>(
		'/2fa',
		'POST',
		request,
		TwoFaChallengeRequestSchema,
		UserWithTokenResponseSchema
	);
};

export const getAllUsers = async (): Promise<UsersResponse> => {
	return await apiFetcher<undefined, UsersResponse>(
		'/',
		'GET',
		undefined,
		undefined,
		UsersResponseSchema
	);
};

export const getFriends = async (): Promise<UsersResponse> => {
	return await apiFetcher<undefined, UsersResponse>(
		'/friends',
		'GET',
		undefined,
		undefined,
		UsersResponseSchema
	);
};

export const addFriend = async (request: AddNewFriendRequest): Promise<void> => {
	await apiFetcher<AddNewFriendRequest, void>(
		'/friends',
		'POST',
		request,
		AddNewFriendRequestSchema,
		undefined
	);
};
