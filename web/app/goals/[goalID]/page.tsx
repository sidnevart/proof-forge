import { GoalDetailScreen } from "@/components/product/goal-detail-screen";

type Props = {
  params: Promise<{ goalID: string }>;
};

export default async function GoalDetailPage({ params }: Props) {
  const { goalID } = await params;
  const id = parseInt(goalID, 10);
  return (
    <div className="page-shell">
      <GoalDetailScreen goalID={id} />
    </div>
  );
}
