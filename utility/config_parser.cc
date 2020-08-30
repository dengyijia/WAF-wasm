#include "config_parser.h"

std::string config_field_to_string(bool include, Keys keys) {
  std::string str = "include: ";
  if (!include) {
    str = "exclude: ";
  }
  for (auto const& s : keys) {
    str += s + ", ";
  }
  return str;
}

std::string Config::to_string() {
  std::string body_str = "\nbody " + config_field_to_string(body_include, body);
  std::string path_str = "\npath " + config_field_to_string(path_include, path);
  std::string header_str = "\nheaders " + config_field_to_string(header_include, headers);
  std::string cookie_str = "\ncookies " + config_field_to_string(cookie_include, cookies);
  return "config: " + content_type + body_str + header_str + cookie_str;
}

/*
 * Validate and store a field in Config
 * Input:
 *   field: a Json object to be parsed, either query param, header, or cookie
 *   include: a pointer to store the parsed result (<field_include in Config)
 *   keys: a pointer to store the parsed result (<field>s in Config)
 * Output:
 *   true on success
 *   false on failure (if both 'include' and 'exclude' are present in the field)
 */
bool validate_config_field(Json field, bool* include, Keys* keys, std::string* log) {
  if (field.is_null()) {
    return true;
  }
  if (!field["include"].is_null() && !field["exclude"].is_null()) {
    *log = "include and exclude cannot both be present";
    return false;
  }
  if (!field["include"].is_null()) {
    *include = true;
    auto include_keys = field["include"].get<Keys>();
    *keys = field["include"].get<Keys>();
  }
  if (!field["exclude"].is_null()) {
    *include = false;
    *keys = field["exclude"].get<Keys>();
  }
  return true;
}

bool parseConfig(std::string configuration, Config* config, std::string* log) {
  // parse configuration string as Json
  Json j = Json::parse(configuration, nullptr, false);
  if (j.is_discarded()) {
    *log = "JSON parse error in configuration";
    return false;
  }

  // validate body param configuration
  auto query_param = j["body"];
  if (!query_param.is_null()) {
    if (query_param["content-type"].is_null()) {
      *log = "missing content-type field under body";
      return false;
    }
    std::string content_type = query_param["content-type"].get<std::string>();
    if (content_type.compare(URLENCODED) != 0) {
      *log =
          ("invalid content type, only application/x-www-form-urlencoded is "
           "supported");
      return false;
    }
    if (!validate_config_field(query_param, &config->body_include, &config->body, log)) {
      return false;
    }
  }

  // validate path param configuration
  if (!validate_config_field(j["path"], &config->path_include, &config->path, log)) {
    return false;
  }

  // validate cookie configuration
  if (!validate_config_field(j["cookie"], &config->cookie_include, &config->cookies, log)) {
    return false;
  }
  // validate header configuration
  if (!validate_config_field(j["header"], &config->header_include, &config->headers, log)) {
    return false;
  }
  *log = "config parsed into context ->" + config->to_string();
  return true;
}
