import Head from "next/head";
import type { GetServerSideProps } from "next";

import { SettingsLayout } from "@/components/SettingsLayout";
import { AdminAccessPanel } from "@/components/settings/AdminAccessPanel";
import {
  fetchAdminSummary,
  fetchViewerAccess,
  requireAdminPage,
  type AuthenticatedPageProps,
  type TenantSummary,
  type ViewerAccount,
  type ViewerInvite,
} from "@/lib/api";

type AdminSettingsProps = {
  invites: ViewerInvite[];
  tenant: TenantSummary;
  viewers: ViewerAccount[];
} & AuthenticatedPageProps;

export default function AdminSettings({ currentUser, invites, tenant, viewers }: AdminSettingsProps) {
  return (
    <>
      <Head>
        <title>Admin - Bifrost</title>
      </Head>
      <SettingsLayout currentUser={currentUser}>
        <AdminAccessPanel invites={invites} tenant={tenant} viewers={viewers} />
      </SettingsLayout>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<AdminSettingsProps> = async (context) => {
  return requireAdminPage(context, async () => {
    const [summary, access] = await Promise.all([
      fetchAdminSummary(context),
      fetchViewerAccess(context),
    ]);

    return {
      invites: access.invites,
      tenant: summary.tenant,
      viewers: access.viewers,
    };
  });
};
