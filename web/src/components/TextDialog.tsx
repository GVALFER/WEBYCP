import { Button, Input, Label, Modal, TextField } from "@heroui/react";
import { type FormEvent, useState } from "react";

type Props = {
  open: boolean;
  title: string;
  label: string;
  value: string;
  pending?: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (value: string) => void;
};

export const TextDialog = ({
  open,
  ...props
}: Props) => (open ? <TextDialogContent {...props} /> : null);

const TextDialogContent = ({
  title,
  label,
  value,
  pending = false,
  onOpenChange,
  onSubmit,
}: Omit<Props, "open">) => {
  const [next, setNext] = useState(value);

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onSubmit(next);
  };

  return (
    <Modal isOpen onOpenChange={onOpenChange}>
      <Modal.Backdrop>
        <Modal.Container placement="center" size="sm">
          <Modal.Dialog>
            <form onSubmit={submit}>
              <Modal.Header>
                <Modal.Heading>{title}</Modal.Heading>
              </Modal.Header>
              <Modal.Body>
                <TextField autoFocus fullWidth isRequired>
                  <Label>{label}</Label>
                  <Input
                    value={next}
                    onChange={(event) => setNext(event.currentTarget.value)}
                  />
                </TextField>
              </Modal.Body>
              <Modal.Footer>
                <Button variant="tertiary" onPress={() => onOpenChange(false)}>
                  Cancel
                </Button>
                <Button type="submit" variant="primary" isDisabled={pending}>
                  Save changes
                </Button>
              </Modal.Footer>
            </form>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  );
};
