import { CheckInScreen } from "@/components/product/checkin-screen";

type Props = {
  params: Promise<{ goalID: string }>;
};

export default async function CheckInPage({ params }: Props) {
  const { goalID } = await params;
  const id = parseInt(goalID, 10);
  return (
    <div className="page-shell">
      <CheckInScreen goalID={id} />
    </div>
  );
}
