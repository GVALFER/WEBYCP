import * as v from "valibot";

export const emailField = v.pipe(
  v.string(),
  v.trim(),
  v.nonEmpty("Enter your email address."),
  v.email("Enter a valid email address."),
);

export const passwordField = v.pipe(
  v.string(),
  v.nonEmpty("Enter your password."),
  v.maxLength(128, "Password must be 128 characters or fewer."),
);

export const nameField = v.pipe(
  v.string(),
  v.trim(),
  v.minLength(2, "Use at least 2 characters."),
  v.maxLength(80, "Use 80 characters or fewer."),
);

export const domainField = v.pipe(
  v.string(),
  v.trim(),
  v.nonEmpty("Enter a hostname."),
  v.maxLength(253, "Use 253 characters or fewer."),
  v.regex(
    /^(?=.{1,253}$)(?:[a-z\d](?:[a-z\d-]{0,61}[a-z\d])?\.)+[a-z\d](?:[a-z\d-]{0,61}[a-z\d])?$/i,
    "Enter a valid hostname.",
  ),
);

export const dbNameField = v.pipe(
  v.string(),
  v.trim(),
  v.nonEmpty("Enter a name."),
  v.maxLength(32, "Use 32 characters or fewer."),
  v.regex(/^[a-zA-Z0-9_]+$/, "Use only letters, numbers and underscores."),
);

export const scheduleField = v.pipe(
  v.string(),
  v.trim(),
  v.maxLength(100, "Use 100 characters or fewer."),
);

export const commandField = v.pipe(
  v.string(),
  v.trim(),
  v.nonEmpty("Enter a command."),
  v.maxLength(1_000, "Use 1000 characters or fewer."),
);

export const issueMessage = (issues: readonly { message: string }[]) =>
  issues[0]?.message ?? "Check the form and try again.";
