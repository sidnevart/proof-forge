import { InviteAcceptScreen } from "@/components/product/invite-accept-screen";

type Props = {
  params: Promise<{ token: string }>;
};

export default async function InvitePage({ params }: Props) {
  const { token } = await params;
  return (
    <div className="page-shell">
      <InviteAcceptScreen token={token} />
    </div>
  );
}
