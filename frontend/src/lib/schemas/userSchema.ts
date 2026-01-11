import * as z from 'zod';

const isAlphaNumericLegalSymbols = (val: string): boolean => /^[a-z0-9,.#$%@^;|_!*&?]+$/i.test(val);

// For user CRUD
const usernameSchema = z
	.string()
	.trim()
	.min(3)
	.max(50)
	.refine((v) => !/\s/.test(v), {
		message: 'Username cannot contain spaces'
	})
	.refine((v) => /^[a-zA-Z0-9._-]+$/.test(v), {
		message: 'Username may only contain letters, numbers, ".", "_" or "-"'
	});

const passwordSchema = z.string().trim().min(3).max(20).refine(isAlphaNumericLegalSymbols, {
	error: 'Password may only contain letters, numbers, and the following symbols: ,.#$%@^;|_!*&?'
});

export const UserSchema = z.object({
	username: usernameSchema,
	email: z.email().trim(),
	avatar: z.url().trim().nullish()
});

export const CreateUserSchema = z.object({
	...UserSchema.shape,
	password: passwordSchema
});

export const CreateUserFormSchema = z
	.object({
		...CreateUserSchema.shape,
		confirmPassword: z.string().trim()
	})
	.refine((data) => data.password === data.confirmPassword, {
		message: 'Passwords do not match',
		path: ['confirmPassword']
	});

export const UpdateUserPasswordRequestSchema = z.object({
	oldPassword: passwordSchema,
	newPassword: passwordSchema
});

export const UpdateUserPasswordFormSchema = z
	.object({
		...UpdateUserPasswordRequestSchema.shape,
		confirmNewPassword: z.string().trim()
	})
	.refine((data) => data.newPassword === data.confirmNewPassword, {
		message: 'New passwords do not match',
		path: ['confirmNewPassword']
	})
	.refine((data) => data.oldPassword !== data.newPassword, {
		message: 'New password must be different from old password',
		path: ['newPassword']
	});

export const LoginUserRequestSchema = z.object({
	username: usernameSchema,
	password: passwordSchema
});

export const LoginUserByEmailRequestSchema = z.object({
	email: z.email().trim(),
	password: passwordSchema
});

export const LoginUserByIdentifierRequestSchema = z.object({
	identifier: z.union([usernameSchema, z.email().trim()]),
	password: passwordSchema
});

const responseAdditionalFields = z.object({
	id: z.int(),
	twoFa: z.boolean(),
	email: z.email().trim(),
	googleOauthId: z.string().trim().nullish(),
	createdAt: z.number()
});

export const UserWithTokenResponseSchema = z.object({
	...UserSchema.shape,
	...responseAdditionalFields.shape,
	token: z.string()
});

export const UpdateUserAvatarFormSchema = z.object({
	avatar: z.url().trim().nullable()
});

export const UpdateUserRequestSchema = UserSchema;

export const UserWithoutTokenResponseSchema = z.object({
	...UserSchema.shape,
	...responseAdditionalFields.shape
});

export const UsernameRequestSchema = z.object({
	username: usernameSchema
});

export const SimpleUserResponseSchema = z.object({
	id: z.int(),
	username: usernameSchema,
	avatar: z.url().trim().nullable()
});

export const UsersResponseSchema = z.array(SimpleUserResponseSchema);

// For Two-factor authentication

// 2FA setup
export const TwoFaSetupResponseSchema = z.object({
	twoFaSecret: z.string(),
	setupToken: z.string(),
	twoFaUri: z.string()
});

export const TwoFaConfirmFormSchema = z.object({
	twoFaCode: z.string().min(6).max(6)
});

// 2FA confirm
export const TwoFaConfirmRequestSchema = z.object({
	twoFaCode: z.string().min(6).max(6),
	setupToken: z.string()
});

// Disable 2FA
export const TwoFaDisableRequestSchema = z.object({
	password: passwordSchema
});

export const TwoFaChallengeRequestSchema = z.object({
	twoFaCode: z.string()
});

export const TwoFaPendingUserResponseSchema = z.object({
	message: z.literal('2FA_REQUIRED'),
	sessionToken: z.string()
});

//
// For Friends
//

export const FriendResponseSchema = z.object({
	...SimpleUserResponseSchema.shape,
	online: z.boolean()
});

export const AddNewFriendRequestSchema = z.object({
	userId: z.int()
});

export const AddNewFriendFormSchema = z.object({
	username: usernameSchema
});

export const GetFriendsResponseSchema = z.array(FriendResponseSchema);
