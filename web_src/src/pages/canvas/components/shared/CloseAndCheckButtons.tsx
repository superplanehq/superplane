import { Button } from "@/components/Button/button";

const CloseAndCheckButtons = ({
  onCancel,
  onConfirm
}: {
  onCancel: () => void;
  onConfirm: () => void;
}) => {
  return (
    <div className="flex justify-end gap-1 pt-2 pb-3">
      <Button
        className="flex items-center border-0"
        outline
        onClick={onCancel}
      >
        <span className="material-symbols-outlined select-none inline-flex items-center justify-center !text-sm" aria-hidden="true">close</span>
      </Button>
      <Button
        className="flex items-center justify-center"
        color="white"
        onClick={onConfirm}
      >
        <span className="material-symbols-outlined select-none inline-flex items-center justify-center !text-sm" aria-hidden="true">check</span>
      </Button>
    </div>
  );
};

export default CloseAndCheckButtons;
