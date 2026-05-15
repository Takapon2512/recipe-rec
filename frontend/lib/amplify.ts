import { Amplify } from "aws-amplify";
// `/auth/callback?code=...` で OAuth コード交換を走らせるための副作用 import。
// これを忘れると Cognito からのリダイレクト後に何も起こらない。
import "aws-amplify/auth/enable-oauth-listener";

Amplify.configure({
  Auth: {
    Cognito: {
      userPoolId: process.env.NEXT_PUBLIC_COGNITO_USER_POOL_ID!,
      userPoolClientId: process.env.NEXT_PUBLIC_COGNITO_CLIENT_ID!,
      loginWith: {
        oauth: {
          domain: process.env.NEXT_PUBLIC_COGNITO_DOMAIN!,
          scopes: ["email", "openid", "profile"],
          redirectSignIn: [process.env.NEXT_PUBLIC_REDIRECT_SIGN_IN!],
          redirectSignOut: [process.env.NEXT_PUBLIC_REDIRECT_SIGN_OUT!],
          responseType: "code",
        },
      },
    },
  },
});
