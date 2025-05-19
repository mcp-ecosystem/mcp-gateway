/**
 * Generate a random string of lowercase letters with specified length
 * @param length - The length of the random string to generate
 * @returns A random string containing only lowercase letters
 */
export function getRandomLetters(length: number): string {
  const letters = 'abcdefghijklmnopqrstuvwxyz';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += letters.charAt(Math.floor(Math.random() * letters.length));
  }
  return result;
}