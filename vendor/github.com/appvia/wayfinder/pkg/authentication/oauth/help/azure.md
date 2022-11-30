You need three pieces of information to configure Wayfinder to use Azure Active Directory authentication:

* **Tenant ID**: the Azure Active Directory tenant to use
* **Client ID**: ID of a client secret for Wayfinder associated with an App Registration in your Azure tenant
* **Client Secret**: secret associated with the Client ID

To configure Azure Active Directory authentication for Wayfinder, you must create an [app 
registration]((https://aka.ms/appregistrations)) via the [Azure portal](https://portal.azure.com/). 
You will then use values from this app registration to configure Wayfinder.

1. Log in to the [Azure portal](https://portal.azure.com/) and search for the 
  **App registrations** service.

2. Select **New registration**.

3. Name it appropriately, for example `Wayfinder`.

4. Under **Support account types**, select **Accounts in this organizational directory only** if 
  you are only allowing access to Wayfinder internally via your organization.

5. Register the application.

6. In the **Authentication** section, **Add a platform** via Platform configurations,
 selecting **Web application**. 

7. Set the following value for **Redirect URIs**:

    * `https://api.wf.yourorg.io/oauth/callback` 

8. Under **Implicit grant and hybrid flows**, enable the **Access token** token issuing tickbox, 
  and then save all changes.

9. Go to the **API permissions** management tab and update the **Microsoft Graph** API to 
  include the following OpenID permissions: **email**, **offline_access** and **profile**.

10. Under the **Certificates & secrets** management tab, create a new client secret with an 
  appropriate description to identify the client secret. 

11. Note the **Value** field of the client secret to use when configuring Wayfinder - this is your **Client Secret**.

12. Note the **Application (client) ID** (this is your **Client ID**) and **Directory (tenant) ID** (this is your **Tenant ID**) 
   values in the overview section of the application.

13. Use the noted tenant ID, client ID, and client secret to configure Wayfinder.

Wayfinder uses the **Email** field defined within an Azure AD user. Therefore, this field must be 
defined within each user in Azure.
