You need three pieces of information to configure Wayfinder to use any OIDC compliant OAuth provider:

* **Server URL**: the base URL for your authentication provider
* **Client ID**: ID of a client secret for Wayfinder
* **Client Secret**: secret associated with the Client ID

In your provider, you must set an **authorized redirect URI** or **callback URL** of:

* `https://api.wf.yourorg.io/oauth/callback` 

If you need advice on integrating your existing single-sign on infrastructure with Wayfinder, please get 
in touch and we will be happy to help out.
