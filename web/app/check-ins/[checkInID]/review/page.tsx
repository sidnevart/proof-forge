import { ApprovalPanel } from "@/components/product/approval-panel";

type Props = {
  params: Promise<{ checkInID: string }>;
};

export default async function ReviewCheckInPage({ params }: Props) {
  const { checkInID } = await params;
  const id = parseInt(checkInID, 10);
  return (
    <div className="page-shell">
      <ApprovalPanel checkInID={id} />
    </div>
  );
}
