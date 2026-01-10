import * as z from 'zod';

import {
	AddNewFriendRequestSchema,
	CreateUserFormSchema,
	CreateUserSchema,
	FriendResponseSchema,
	GetFriendsResponseSchema,
	LoginUserByEmailRequestSchema,
	LoginUserByIdentifierRequestSchema,
	LoginUserRequestSchema,
	SimpleUserResponseSchema,
	TwoFaChallengeRequestSchema,
	TwoFaConfirmRequestSchema,
	TwoFaDisableRequestSchema,
	TwoFaPendingUserResponseSchema,
	TwoFaSetupResponseSchema,
	UpdateUserPasswordRequestSchema,
	UpdateUserRequestSchema,
	UsernameRequestSchema,
	UserSchema,
	UsersResponseSchema,
	UserWithoutTokenResponseSchema,
	UserWithTokenResponseSchema
} from './userSchema.js';

export type User = z.infer<typeof UserSchema>;
export type CreateUserRequest = z.infer<typeof CreateUserSchema>;
export type CreateUserForm = z.infer<typeof CreateUserFormSchema>;
export type UpdateUserPasswordRequest = z.infer<typeof UpdateUserPasswordRequestSchema>;
export type LoginUserRequest = z.infer<typeof LoginUserRequestSchema>;
export type LoginUserByEmailRequest = z.infer<typeof LoginUserByEmailRequestSchema>;
export type LoginUserByIdentifierRequest = z.infer<typeof LoginUserByIdentifierRequestSchema>;

export type TwoFaChallengeRequest = z.infer<typeof TwoFaChallengeRequestSchema>;
export type TwoFaConfirmRequest = z.infer<typeof TwoFaConfirmRequestSchema>;
export type UserWithTokenResponse = z.infer<typeof UserWithTokenResponseSchema>;
export type TwoFaDisableRequest = z.infer<typeof TwoFaDisableRequestSchema>;
export type TwoFaSetupResponse = z.infer<typeof TwoFaSetupResponseSchema>;
export type TwoFaPendingUserResponse = z.infer<typeof TwoFaPendingUserResponseSchema>;

export type UpdateUserRequest = z.infer<typeof UpdateUserRequestSchema>;
export type UserWithoutTokenResponse = z.infer<typeof UserWithoutTokenResponseSchema>;

export type UsersResponse = z.infer<typeof UsersResponseSchema>;

export type SimpleUserResponse = z.infer<typeof SimpleUserResponseSchema>;
export type UsernameRequest = z.infer<typeof UsernameRequestSchema>;

export type GetFriendsResponse = z.infer<typeof GetFriendsResponseSchema>;
export type AddNewFriendRequest = z.infer<typeof AddNewFriendRequestSchema>;
export type FriendResponse = z.infer<typeof FriendResponseSchema>;
