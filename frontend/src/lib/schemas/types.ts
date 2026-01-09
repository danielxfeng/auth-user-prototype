import * as z from 'zod';

import {
	AddNewFriendRequestSchema,
	CreateUserSchema,
	ErrorResponseSchema,
	FriendResponseSchema,
	GetFriendsResponseSchema,
	GoogleOAuthCallbackSchema,
	JWTPayloadSchema,
	LoginUserByEmailRequestSchema,
	LoginUserByIdentifierRequestSchema,
	LoginUserRequestSchema,
	OauthStateJwtPayloadSchema,
	SetTwoFaRequestSchema,
	SimpleUserResponseSchema,
	TwoFaChallengeRequestSchema,
	TwoFaJwtPayloadSchema,
	TwoFaSetupJwtPayloadSchema,
	TwoFaSetupResponseSchema,
	UpdateUserPasswordRequestSchema,
	UpdateUserRequestSchema,
	UserJwtPayloadSchema,
	UsernameRequestSchema,
	UserSchema,
	UsersResponseSchema,
	UserValidationResponseSchema,
	UserWithoutTokenResponseSchema,
	UserWithTokenOptionalTwoFaResponseSchema,
	UserWithTokenResponseSchema
} from './userSchema.js';

export type User = z.infer<typeof UserSchema>;
export type CreateUserRequest = z.infer<typeof CreateUserSchema>;
export type UpdateUserPasswordRequest = z.infer<typeof UpdateUserPasswordRequestSchema>;
export type LoginUserRequest = z.infer<typeof LoginUserRequestSchema>;
export type LoginUserByEmailRequest = z.infer<typeof LoginUserByEmailRequestSchema>;
export type LoginUserByIdentifierRequest = z.infer<typeof LoginUserByIdentifierRequestSchema>;

export type UserWithTokenOptionalTwoFaResponse = z.infer<
	typeof UserWithTokenOptionalTwoFaResponseSchema
>;

export type TwoFaChallengeRequest = z.infer<typeof TwoFaChallengeRequestSchema>;
export type UserWithTokenResponse = z.infer<typeof UserWithTokenResponseSchema>;
export type SetTwoFaRequest = z.infer<typeof SetTwoFaRequestSchema>;
export type TwoFaSetupResponse = z.infer<typeof TwoFaSetupResponseSchema>;

export type UpdateUserRequest = z.infer<typeof UpdateUserRequestSchema>;
export type UserWithoutTokenResponse = z.infer<typeof UserWithoutTokenResponseSchema>;

export type UsersResponse = z.infer<typeof UsersResponseSchema>;

export type SimpleUserResponse = z.infer<typeof SimpleUserResponseSchema>;
export type UsernameRequest = z.infer<typeof UsernameRequestSchema>;

export type GetFriendsResponse = z.infer<typeof GetFriendsResponseSchema>;
export type AddNewFriendRequest = z.infer<typeof AddNewFriendRequestSchema>;
export type FriendResponse = z.infer<typeof FriendResponseSchema>;

export type UserJwtPayload = z.infer<typeof UserJwtPayloadSchema>;

export type TwoFaJwtPayload = z.infer<typeof TwoFaJwtPayloadSchema>;
export type TwoFaSetupJwtPayload = z.infer<typeof TwoFaSetupJwtPayloadSchema>;

export type OauthStateJwtPayload = z.infer<typeof OauthStateJwtPayloadSchema>;
export type JWTPayload = z.infer<typeof JWTPayloadSchema>;

export type UserValidationResponse = z.infer<typeof UserValidationResponseSchema>;

export type ErrorResponse = z.infer<typeof ErrorResponseSchema>;

export type GoogleOAuthCallback = z.infer<typeof GoogleOAuthCallbackSchema>;
