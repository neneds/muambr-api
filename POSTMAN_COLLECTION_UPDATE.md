# Postman Collection Update Summary

## Overview
Updated the Postman collection to include comprehensive testing for the new `useMacroRegion` parameter feature.

## New Requests Added

### 1. "With Macro Region (EU)"
- **URL**: `/api/v1/comparisons/search?name=iphone 15&baseCountry=PT&currentUserCountry=ES&useMacroRegion=true&currency=EUR&limit=5`
- **Purpose**: Demonstrates macro region functionality within EU
- **Expected Behavior**: Returns results from Portugal (base) + all EU countries (PT, ES, DE, GB)
- **Key Feature**: Shows how PT extractors don't duplicate even though PT is both base country and in EU macro region

### 2. "With Macro Region Disabled (Default)"
- **URL**: `/api/v1/comparisons/search?name=iphone 15&baseCountry=PT&currentUserCountry=ES&useMacroRegion=false&currency=EUR&limit=5`
- **Purpose**: Shows the default behavior (backward compatibility)
- **Expected Behavior**: Returns results from Portugal (base) + Spain (current) only
- **Key Feature**: Demonstrates that existing behavior is preserved when macro region is disabled

### 3. "Cross-Region Example (Brazil to EU)"
- **URL**: `/api/v1/comparisons/search?name=samsung galaxy&baseCountry=BR&currentUserCountry=DE&useMacroRegion=true&currency=BRL&limit=3`
- **Purpose**: Demonstrates cross-macro-region scenario
- **Expected Behavior**: Returns results from Brazil (LATAM base) + all EU countries (PT, ES, DE, GB)
- **Key Feature**: Shows currency conversion from EUR to BRL for EU results

## Updated Collection Description
Enhanced the collection description to include:
- Feature overview of the new `useMacroRegion` parameter
- Explanation of macro region mappings:
  - **EU**: Portugal, Spain, Germany, United Kingdom
  - **NA**: United States  
  - **LATAM**: Brazil
- Deduplication behavior explanation

## Response Examples
Each new request includes realistic sample responses showing:
- ✅ **Proper country attribution**: Each product shows its origin country
- ✅ **Currency conversion**: When target currency differs from product currency
- ✅ **Store naming**: Consistent "StoreName - CountryCode" format
- ✅ **No duplicates**: Demonstrates deduplication working correctly

## Parameter Testing Coverage

| Parameter | Test Coverage |
|-----------|--------------|
| `useMacroRegion=true` | ✅ EU macro region example |
| `useMacroRegion=false` | ✅ Disabled/default behavior |
| Cross-region scenarios | ✅ Brazil base + EU macro region |
| Currency conversion | ✅ EUR→BRL in cross-region example |
| Backward compatibility | ✅ Existing requests unchanged |

## Validation
- ✅ JSON syntax validation passed
- ✅ All existing requests preserved 
- ✅ New requests follow consistent structure
- ✅ Response examples match API schema
- ✅ Parameter combinations cover main use cases

## Usage Instructions

### Testing Macro Region Feature
1. Import the updated collection into Postman
2. Set the `base_url` variable to your API endpoint
3. Run the "With Macro Region (EU)" request to see expanded results
4. Compare with "With Macro Region Disabled" to see the difference
5. Try "Cross-Region Example" to test currency conversion scenarios

### Key Test Scenarios
- **European Traveler**: Portuguese user in Spain → Gets results from all EU
- **Cross-Region Business**: Brazilian company checking EU prices → Gets BRL-converted results
- **Backward Compatibility**: Existing integrations → Work unchanged with `useMacroRegion=false`

The collection now provides comprehensive testing coverage for both the new macro region functionality and maintains full backward compatibility with existing API behavior.