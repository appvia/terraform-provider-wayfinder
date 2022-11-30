You need four pieces of information to configure Wayfinder to use Auth0 authentication:

* **Tenant**: the Auth0 tenant to use
* **Region**: the Auth0 region in which your tenant exists
* **Client ID**: ID of a specific application added to your Auth0 tenant for Wayfinder to use
* **Client Secret**: secret associated with the Client ID

To add a new application to Auth0 for Wayfinder:

1. If you don't have one, sign up for an account from the [Auth0 home page](https://auth0.com).

2. From the drop-down in the top left of the Auth0 dashboard note your auth0 **Tenant** (e.g. 'my-org') 
   and **Region** (e.g. 'eu' or 'us').

3. From the dashboard side menu choose **Applications**, and then **Create Application**.

4. Give the application a name (e.g. 'wayfinder') and choose **Regular Web Application**.

5. Once provisioned click the **Settings** tab.

6. Note the **Client ID** and **Client Secret** for your new application.

7. Scroll down to **Allowed Callback URLs** and add:

    * `https://api.wf.yourorg.io/oauth/callback` 

8. Click the down arrow next to **Advanced Settings**.

9. Click the **OAuth** tab and ensure that:

   * **JsonWebToken Signature Algorithm** is set to RS256
   * **OIDC Conformant** is ON.

10. Use the noted tenant, region, client ID, and client secret to configure Wayfinder.
