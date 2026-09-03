import { errorInfo } from "reqly-js";

export const errorMessage = async (error: unknown) => {
  const info = await errorInfo(error);
  return info.message;
};
