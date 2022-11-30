You need two pieces of information to configure Wayfinder to use Google Identity/Google Workspace authentication:

* **Client ID**: ID of an OAuth 2.0 Client ID for Wayfinder in your Google account
* **Client Secret**: secret associated with the Client ID

To configure Google Identity/Google Workspace (GSuite) authentication for Wayfinder, you must add an
OAuth 2.0 Client ID using the [Google Cloud Platform console](https://console.cloud.google.com/). 
You then use the values from this Client ID to configure Wayfinder.

1. Log in to the [Google Cloud Platform console](https://console.cloud.google.com/) and open the 
  **APIs & Services** section.

2. Select **Credentials**, and then **Create Credentials > OAuth client ID**.

3. Name it appropriately, for example, `wf`.

4. Add an authorized redirect URI:

    * `https://api.wf.yourorg.io/oauth/callback` 

5. Create the credential. 

  Note the **Client ID** and **Client Secret**.

6. Go to the **OAuth consent screen** section, give your app a name (this will be shown to users 
  when they log in), and **set the User type to Internal**. This ensures that only users in your 
  organization will be able to log in using this method.

7. Use the noted client ID, and client secret to configure Wayfinder.
