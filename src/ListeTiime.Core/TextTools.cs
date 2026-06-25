using System.Globalization;
using System.Text;
using System.Text.RegularExpressions;

namespace ListeTiime.Core;

public static class TextTools
{
    private static readonly Regex GrandLivreFileNameRegex = new(
        @"^grand_livre_(\d{4})-\d{2}-\d{2}_(\d{4})-\d{2}-\d{2}_(.+?)(?: \(\d+\))?\.csv$",
        RegexOptions.IgnoreCase | RegexOptions.Compiled);

    public static string NormalizeHeader(string value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return string.Empty;
        }

        var decomposed = value.Normalize(NormalizationForm.FormD);
        var builder = new StringBuilder(decomposed.Length);
        var previousUnderscore = false;

        foreach (var character in decomposed)
        {
            var category = CharUnicodeInfo.GetUnicodeCategory(character);
            if (category == UnicodeCategory.NonSpacingMark)
            {
                continue;
            }

            var lower = char.ToLowerInvariant(character);
            if (char.IsAsciiLetterOrDigit(lower))
            {
                builder.Append(lower);
                previousUnderscore = false;
            }
            else if (!previousUnderscore)
            {
                builder.Append('_');
                previousUnderscore = true;
            }
        }

        return builder.ToString().Trim('_');
    }

    public static decimal ParseAmount(string value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return 0m;
        }

        var cleaned = value
            .Replace('\u00A0', ' ')
            .Replace("€", string.Empty, StringComparison.Ordinal)
            .Trim();

        if (decimal.TryParse(
                cleaned,
                NumberStyles.Number | NumberStyles.AllowCurrencySymbol | NumberStyles.AllowLeadingSign,
                AppConstants.FrenchCulture,
                out var frenchAmount))
        {
            return frenchAmount;
        }

        cleaned = cleaned
            .Replace(" ", string.Empty, StringComparison.Ordinal)
            .Replace(",", ".", StringComparison.Ordinal);

        if (decimal.TryParse(
                cleaned,
                NumberStyles.Number | NumberStyles.AllowLeadingSign,
                CultureInfo.InvariantCulture,
                out var invariantAmount))
        {
            return invariantAmount;
        }

        throw new FormatException($"Montant invalide : {value}");
    }

    public static DateTime? ParseDate(string value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return null;
        }

        var formats = new[] { "dd/MM/yyyy", "yyyy-MM-dd", "dd-MM-yyyy" };
        if (DateTime.TryParseExact(
                value.Trim(),
                formats,
                AppConstants.FrenchCulture,
                DateTimeStyles.None,
                out var date))
        {
            return date;
        }

        throw new FormatException($"Date invalide : {value}");
    }

    public static decimal? NullIfZero(decimal amount) => amount == 0m ? null : amount;

    public static string FormatAmount(decimal amount)
    {
        return amount
            .ToString("#,##0.00", AppConstants.FrenchCulture)
            .Replace('\u00A0', ' ')
            .Replace('\u202F', ' ');
    }

    public static FileMetadata? TryParseGrandLivreMetadata(string path)
    {
        var name = Path.GetFileName(path);
        var match = GrandLivreFileNameRegex.Match(name);
        if (!match.Success)
        {
            return null;
        }

        var startYear = match.Groups[1].Value;
        var endYear = match.Groups[2].Value;
        var year = startYear == endYear ? startYear : $"{startYear}-{endYear}";
        var client = match.Groups[3].Value
            .Replace('_', ' ')
            .Replace('-', ' ')
            .Trim()
            .ToUpperInvariant();

        return new FileMetadata(client, year);
    }

    public static string SanitizeWorksheetName(string name)
    {
        var invalid = new HashSet<char>(Path.GetInvalidFileNameChars()) { '[', ']', '*', '?', '/', '\\', ':' };
        var cleaned = new string(name.Select(character => invalid.Contains(character) ? ' ' : character).ToArray()).Trim();
        return cleaned.Length <= 31 ? cleaned : cleaned[..31];
    }
}
