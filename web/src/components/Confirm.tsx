import { AlertDialog, Button } from "@heroui/react";
import { TriangleAlert } from "lucide-react";
import type { ReactNode } from "react";

type Props = {
  children: ReactNode;
  title: string;
  description: string;
  action?: string;
  onConfirm: () => void;
};

export const Confirm = ({
  children,
  title,
  description,
  action = "Confirm",
  onConfirm,
}: Props) => (
  <AlertDialog>
    <AlertDialog.Trigger>{children}</AlertDialog.Trigger>
    <AlertDialog.Backdrop>
      <AlertDialog.Container placement="center" size="sm">
        <AlertDialog.Dialog>
          {({ close }) => (
            <>
              <AlertDialog.Header>
                <AlertDialog.Icon status="danger">
                  <TriangleAlert aria-hidden="true" />
                </AlertDialog.Icon>
                <AlertDialog.Heading>{title}</AlertDialog.Heading>
              </AlertDialog.Header>
              <AlertDialog.Body>{description}</AlertDialog.Body>
              <AlertDialog.Footer>
                <Button variant="tertiary" onPress={close}>
                  Cancel
                </Button>
                <Button
                  variant="danger"
                  onPress={() => {
                    close();
                    onConfirm();
                  }}
                >
                  {action}
                </Button>
              </AlertDialog.Footer>
            </>
          )}
        </AlertDialog.Dialog>
      </AlertDialog.Container>
    </AlertDialog.Backdrop>
  </AlertDialog>
);
