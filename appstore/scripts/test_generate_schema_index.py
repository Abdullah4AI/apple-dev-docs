#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import json
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("generate-schema-index.py")
SPEC = importlib.util.spec_from_file_location("generate_schema_index", MODULE_PATH)
assert SPEC and SPEC.loader
generate_schema_index = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(generate_schema_index)


class GenerateSchemaIndexTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.spec = json.loads(generate_schema_index.SPEC_PATH.read_text())
        cls.schemas = cls.spec["components"]["schemas"]

    def test_extracts_required_to_one_relationship(self) -> None:
        relationships = generate_schema_index.extract_request_relationships(
            self.schemas,
            "#/components/schemas/InAppPurchaseVersionCreateRequest",
        )
        self.assertEqual(
            relationships,
            {
                "inAppPurchase": {
                    "resourceType": "inAppPurchases",
                    "cardinality": "one",
                    "required": True,
                }
            },
        )

    def test_extracts_required_and_optional_relationships(self) -> None:
        relationships = generate_schema_index.extract_request_relationships(
            self.schemas,
            "#/components/schemas/ProfileCreateRequest",
        )
        self.assertEqual(
            relationships["certificates"],
            {
                "resourceType": "certificates",
                "cardinality": "many",
                "required": True,
            },
        )
        self.assertEqual(
            relationships["devices"],
            {
                "resourceType": "devices",
                "cardinality": "many",
                "required": False,
            },
        )

    def test_build_index_includes_review_submission_version_relationships(self) -> None:
        endpoints = generate_schema_index.build_index(self.spec)
        endpoint = next(
            endpoint
            for endpoint in endpoints
            if endpoint["method"] == "POST"
            and endpoint["path"] == "/v1/reviewSubmissionItems"
        )
        self.assertEqual(
            endpoint["requestRelationships"]["subscriptionVersion"],
            {
                "resourceType": "subscriptionVersions",
                "cardinality": "one",
                "required": False,
            },
        )
        self.assertTrue(
            endpoint["requestRelationships"]["reviewSubmission"]["required"]
        )

    def test_build_index_includes_action_for_get_cardinality(self) -> None:
        endpoints = generate_schema_index.build_index(self.spec)
        endpoint = next(
            endpoint
            for endpoint in endpoints
            if endpoint["method"] == "GET"
            and endpoint["path"] == "/v1/builds/{id}/appStoreVersion"
        )
        self.assertEqual(endpoint["getAction"], "get")

    def test_build_index_classifies_array_valued_metrics_as_list(self) -> None:
        endpoints = generate_schema_index.build_index(self.spec)
        endpoint = next(
            endpoint
            for endpoint in endpoints
            if endpoint["method"] == "GET"
            and endpoint["path"] == "/v1/apps/{id}/metrics/betaTesterUsages"
        )
        self.assertEqual(endpoint["getAction"], "list")

    def test_get_operation_action_falls_back_for_unresolved_data_ref(self) -> None:
        schemas = {
            "CollectionResponse": {
                "properties": {
                    "data": {"$ref": "#/components/schemas/MissingResource"}
                }
            }
        }
        action = generate_schema_index.get_operation_action(
            schemas,
            "widgets_getCollection",
            "#/components/schemas/CollectionResponse",
        )
        self.assertEqual(action, "list")

    def test_every_get_operation_has_an_indexed_action(self) -> None:
        endpoints = generate_schema_index.build_index(self.spec)
        get_endpoints = [
            endpoint for endpoint in endpoints if endpoint["method"] == "GET"
        ]
        self.assertGreater(len(get_endpoints), 0)
        self.assertTrue(
            all(
                endpoint.get("getAction") in {"get", "list"}
                for endpoint in get_endpoints
            )
        )

    def test_every_request_relationship_in_snapshot_is_indexed(self) -> None:
        endpoints = generate_schema_index.build_index(self.spec)
        indexed = {
            (endpoint["method"], endpoint["path"]): endpoint.get(
                "requestRelationships", {}
            )
            for endpoint in endpoints
        }

        checked = 0
        for path, operations in self.spec["paths"].items():
            for method, operation in operations.items():
                if method not in {"get", "post", "patch", "delete"}:
                    continue
                request_schema = (
                    operation.get("requestBody", {})
                    .get("content", {})
                    .get("application/json", {})
                    .get("schema", {})
                )
                ref = request_schema.get("$ref")
                if not ref:
                    continue
                resolved = generate_schema_index.resolve_ref(self.schemas, ref)
                data = resolved.get("properties", {}).get("data", {})
                if "$ref" in data:
                    data = (
                        generate_schema_index.resolve_ref(self.schemas, data["$ref"])
                        or data
                    )
                relationships = data.get("properties", {}).get("relationships", {})
                if "$ref" in relationships:
                    relationships = (
                        generate_schema_index.resolve_ref(
                            self.schemas, relationships["$ref"]
                        )
                        or relationships
                    )
                required = set(relationships.get("required", []))
                expected = {}
                for name, relationship in relationships.get("properties", {}).items():
                    if "$ref" in relationship:
                        relationship = (
                            generate_schema_index.resolve_ref(
                                self.schemas, relationship["$ref"]
                            )
                            or relationship
                        )
                    linkage = relationship.get("properties", {}).get("data", {})
                    if "$ref" in linkage:
                        linkage = (
                            generate_schema_index.resolve_ref(
                                self.schemas, linkage["$ref"]
                            )
                            or linkage
                        )
                    cardinality = (
                        "many" if linkage.get("type") == "array" else "one"
                    )
                    identifier = (
                        linkage.get("items", {})
                        if cardinality == "many"
                        else linkage
                    )
                    if "$ref" in identifier:
                        identifier = (
                            generate_schema_index.resolve_ref(
                                self.schemas, identifier["$ref"]
                            )
                            or identifier
                        )
                    resource_types = (
                        identifier.get("properties", {})
                        .get("type", {})
                        .get("enum", [])
                    )
                    self.assertEqual(len(resource_types), 1)
                    expected[name] = {
                        "resourceType": resource_types[0],
                        "cardinality": cardinality,
                        "required": name in required,
                    }
                if not expected:
                    continue

                checked += len(expected)
                self.assertEqual(indexed[(method.upper(), path)], expected)

        self.assertGreater(checked, 0)


if __name__ == "__main__":
    unittest.main()
